package canticles

import (
	"errors"
	"io/ioutil"
	"os"
	"path"
	"regexp"
	"testing"

	"golang.org/x/tools/go/vcs"
)

var errTest = errors.New("Test err")

func TestDefaultRepoResolver(t *testing.T) {
	dr := &DefaultRepoResolver{os.ExpandEnv("$GOPATH")}
	// Try a VCS resolution against someone supports go get syntax
	importPath := "golang.org/x/tools/go/vcs"
	vcs, err := dr.ResolveRepo(importPath, nil)
	if err != nil {
		t.Errorf("DefaultRepoResolver returned error for golang.org repo: %s", err.Error())
	}
	if vcs == nil {
		t.Fatalf("DefaultRepoResolver returned nil vcs for repo: %s", importPath)
	}

	v := vcs.(*PackageVCS)
	if v.Repo.Root != "golang.org/x/tools" {
		t.Errorf("DefaultRepoResolver did not set correct root for repo got %s expected %s", v.Repo.Root, "golang.org/x/tools")
	}
	if v.Repo.Repo == "" {
		t.Errorf("DefaultRepoResolver did not set any repo for repo %s", importPath)
	}
}

func TestRemoteRepoResolver(t *testing.T) {
	rr := &RemoteRepoResolver{os.ExpandEnv("$GOPATH")}

	dep := &CanticleDependency{
		SourcePath: "git@github.com:Comcast/Canticle.git",
		Root:       "github.com/Comcast/Canticle",
	}

	vcs, err := rr.ResolveRepo(dep.Root, dep)
	if err != nil {
		t.Errorf("RemoteRepoResolver returned error for our own repo: %s", err.Error())
	}
	if vcs == nil {
		t.Fatalf("RemoteRepoResolverResolveRepo returned nil vcs for repo: %+v", dep)
	}
	v := vcs.(*PackageVCS)
	expectedRoot := "github.com/Comcast/Canticle"
	if v.Repo.Root != expectedRoot {
		t.Errorf("RemoteRepoResolver did not set correct root for repo got %s expected %s", v.Repo.Root, expectedRoot)
	}
	expectedURL := "git@github.com:Comcast/Canticle.git"
	if v.Repo.Repo != expectedURL {
		t.Errorf("ResolveRepo did not set correct repo for repo got %s expected %s", v.Repo.Repo, expectedURL)
	}

	// Try a VCS resolution that just flat fails
	dep = &CanticleDependency{
		Root:       "nothere.comcast.com/viper-cog/cant",
		SourcePath: "git@nothere.comcast.com:viper-cog/cant.git",
	}
	vcs, err = rr.ResolveRepo(dep.Root, dep)
	if err == nil {
		t.Errorf("RemoteRepoResolver returned no error for a package that does not exist")
	}
	if vcs != nil {
		t.Errorf("RemoteRepoResolver returned non nil vcs for repo: %+v", dep)
	}
}

func TestLocalRepoResolver(t *testing.T) {
	gopath, err := EnvGoPath()
	if err != nil {
		t.Fatalf("Could not get gopath: %s", err.Error())
	}
	lr := &LocalRepoResolver{
		LocalPath: gopath,
	}

	pkg := "github.com/Comcast/Canticle"
	vcs, err := lr.ResolveRepo(pkg, nil)
	if err != nil {
		t.Errorf("LocalRepoResolver returned error resolving our own package %s", err.Error())
	}

	if vcs == nil {
		t.Fatalf("LocalRepoResolver returned a nil VCS resolving our own package")
	}
	v := vcs.(*LocalVCS)
	if v.Cmd.Cmd != "git" {
		t.Errorf("LocalRepoResolver did not set correct vcs command %s expected %s", v.Cmd.Cmd, "git")
	}

	// Test dealing with a package whose vcs root != the importpath
	pkg = "golang.org/x/tools/go/vcs"
	vcs, err = lr.ResolveRepo(pkg, nil)
	if err != nil {
		t.Errorf("LocalRepoResolver returned error resolving our own package %s", err.Error())
	}

	if vcs == nil {
		t.Fatalf("LocalRepoResolver returned a nil VCS resolving our own package")
	}
	v = vcs.(*LocalVCS)
	if v.Cmd.Cmd != "git" {
		t.Errorf("LocalRepoResolver did not set correct vcs command %s expected %s", v.Cmd.Cmd, "git")
	}
}

func TestResolveRootWithNoSlash(t *testing.T) {
	gopath, err := EnvGoPath()
	if err != nil {
		t.Fatalf("Could not get gopath: %s", err.Error())
	}

	dep := &CanticleDependency{
		Root:       "camlistore.org",
		SourcePath: "https://camlistore.googlesource.com/camlistore",
	}

	dr := &DefaultRepoResolver{gopath}
	_, err = dr.ResolveRepo(dep.Root, dep)
	if err != nil {
		t.Errorf("DefaultRepoResolver could not resolve Root that does not contain a slash: %v", err)
	}

	rr := &RemoteRepoResolver{gopath}
	_, err = rr.ResolveRepo(dep.Root, dep)
	if err != nil {
		t.Errorf("RemoteRepoResolver could not resolve Root that does not contain a slash: %v", err)
	}

	// Create a temp test dir for testing no slash local
	testHome, err := ioutil.TempDir("", "cant-test-src")
	if err != nil {
		t.Fatalf("Error creating tempdir: %s", err.Error())
	}
	defer os.RemoveAll(testHome)
	if err := os.MkdirAll(path.Join(testHome, "src", "camlistore.org", ".git"), 0755); err != nil {
		t.Fatalf("Error creating tempdir: %s", err.Error())
	}

	lr := &LocalRepoResolver{testHome}
	_, err = lr.ResolveRepo(dep.Root, dep)
	if err != nil {
		t.Errorf("LocalRepoResolver could not resolve Root that does not contain a slash: %v", err)
	}

}

type TestVCS struct {
	Updated int
	Created int
	Err     error
	Rev     string
	Source  string
	Root    string
}

func (v *TestVCS) UpdateBranch(branch string) (bool, string, error) {
	return false, "", nil
}

func (v *TestVCS) Create(rev string) error {
	v.Rev = rev
	v.Created++
	return v.Err
}

func (v *TestVCS) SetRev(rev string) error {
	v.Rev = rev
	v.Updated++
	return v.Err
}

func (v *TestVCS) GetRev() (string, error) {
	return v.Rev, v.Err
}

func (v *TestVCS) GetSource() (string, error) {
	return v.Source, v.Err
}

func (v *TestVCS) GetRoot() string {
	return v.Root
}

func (v *TestVCS) GetBranch() (string, error) {
	return v.Rev, v.Err
}

type testResolve struct {
	path string
	dep  *CanticleDependency
}

type testResolver struct {
	resolutions []testResolve
	response    []resolve
}

func (tr *testResolver) ResolveRepo(i string, d *CanticleDependency) (VCS, error) {
	tr.resolutions = append(tr.resolutions, testResolve{i, d})
	resp := tr.response[0]
	tr.response = tr.response[1:]
	return resp.v, resp.err
}

func TestCompositeRepoResolver(t *testing.T) {
	res := &TestVCS{}
	tr1 := &testResolver{response: []resolve{{nil, errTest}}}
	tr2 := &testResolver{response: []resolve{{res, nil}}}

	cr := &CompositeRepoResolver{[]RepoResolver{tr1, tr2}}

	dep := &CanticleDependency{
		Root: "testi",
	}
	v, err := cr.ResolveRepo(dep.Root, dep)
	if err != nil {
		t.Errorf("CompositeRepoResolver returned error with valid resolve %s", err.Error())
	}
	if v != res {
		t.Errorf("CompositeRepoResolver returned incorrect vcs")
	}
	if tr1.resolutions[0].path != dep.Root {
		t.Errorf("CompositeRepoResolver tr1 bad import path")
	}
	if tr1.resolutions[0].dep != dep {
		t.Errorf("CompositeRepoResolver tr1 bad dep")
	}
	if tr2.resolutions[0].dep != dep {
		t.Errorf("CompositeRepoResolver tr2 bad dep")
	}

	tr1 = &testResolver{response: []resolve{{nil, errTest}}}
	tr2 = &testResolver{response: []resolve{{nil, errTest}}}
	cr = &CompositeRepoResolver{[]RepoResolver{tr1, tr2}}
	v, err = cr.ResolveRepo(dep.Root, dep)
	if re := ResolutionFailureErr(err); re == nil {
		t.Errorf("CompositeRepoResolver did not return resolution failure")
	}
}

func TestMemoizedRepoResolver(t *testing.T) {
	res := &TestVCS{}
	tr1 := &testResolver{response: []resolve{{res, nil}}}
	mr := NewMemoizedRepoResolver(tr1)
	dep := &CanticleDependency{
		Root: "testi",
	}
	v, err := mr.ResolveRepo(dep.Root, dep)
	if err != nil {
		t.Errorf("MemoizedRepoResolver returned error %s", err.Error())
	}
	if v != res {
		t.Errorf("MemoizedRepoResolver returned wrong vcs")
	}
	if len(tr1.resolutions) != 1 {
		t.Errorf("MemoizedRepoResolver did not call tr1 only once")
	}

	v, err = mr.ResolveRepo(dep.Root, dep)
	if err != nil {
		t.Errorf("MemoizedRepoResolver returned error %s", err.Error())
	}
	if v != res {
		t.Errorf("MemoizedRepoResolver returned wrong vcs")
	}
	if len(tr1.resolutions) != 1 {
		t.Errorf("MemoizedRepoResolver did not call tr1 only once")
	}
}

var (
	expectedRev = "testrev"
	TestRevCmd  = &VCSCmd{
		Name:       "Test",
		Cmd:        "echo",
		Args:       []string{expectedRev},
		ParseRegex: regexp.MustCompile(`^(\S+)$`),
	}
)

// TODO: Add coverage for LocalVCS and PackageVCS
func TestVCSCmds(t *testing.T) {

	testHome, err := ioutil.TempDir("", "cant-test")
	if err != nil {
		t.Fatalf("Error creating tempdir: %s", err.Error())
	}
	defer os.RemoveAll(testHome)

	rev, err := TestRevCmd.Exec(testHome)
	if err != nil {
		t.Fatalf("Error running valid test exec command: %s", err.Error())
	}
	if rev != expectedRev {
		t.Errorf("Exec not %s not match expected %s", rev, expectedRev)
	}

	rev, err = TestRevCmd.Exec("someinvaliddir")
	if err == nil {
		t.Fatalf("No Error running invalid test exec command")
	}
	if rev != "" {
		t.Errorf("Rev returned non empty string for errored command")
	}

	TestRevCmd.Args = []string{"this should be invalid"}
	rev, err = TestRevCmd.Exec(testHome)
	if err == nil {
		t.Fatalf("No Error running invalid test exec command")
	}
	if rev != "" {
		t.Errorf("Exec returned non empty string for regex that did not match")
	}
	TestRevCmd.Args = []string{expectedRev}
}

var (
	TestVCSCmd = &vcs.Cmd{
		Name:        "Test",
		Cmd:         "echo",
		CreateCmd:   "create",
		DownloadCmd: "download",
		TagCmd: []vcs.TagCmd{
			{"show-ref", `(?:tags|origin)/(\S+)$`},
		},
		TagLookupCmd: []vcs.TagCmd{
			{"-n {tag}", `(\S+)$`},
		},
		TagSyncCmd:     "checkout {tag}",
		TagSyncDefault: "checkout master",

		Scheme:  []string{"test", "https"},
		PingCmd: "ping {scheme}://{repo}",
	}
)

func TestLocalVCS(t *testing.T) {
	vcs.Verbose = true
	testHome, err := ioutil.TempDir("", "cant-test-src")
	if err != nil {
		t.Fatalf("Error creating tempdir: %s", err.Error())
	}
	defer os.RemoveAll(testHome)

	pkgname := "test.com/test"
	childpkg := "test.com/test/child"
	if err := os.MkdirAll(PackageSource(testHome, childpkg), 0755); err != nil {
		t.Fatalf("Error creating tempdir: %s", err.Error())
	}

	v := NewLocalVCS(childpkg, pkgname, testHome, TestVCSCmd)
	rev, err := v.GetRev()
	if err != nil {
		t.Fatalf("Local vcs should not return error with no rev command")
	}
	if rev != "" {
		t.Errorf("Rev returned non empty string for no rev command: %s %+v", rev, v)
	}

	RevCmds[TestRevCmd.Name] = TestRevCmd
	v = NewLocalVCS(childpkg, pkgname, testHome, TestVCSCmd)
	v.Branches = func(path string) ([]string, error) {
		return []string{"testrev"}, nil
	}
	rev, err = v.GetRev()
	if err != nil {
		t.Errorf("Error getting valid rev: %s", err.Error())
	}
	if rev != expectedRev {
		t.Errorf("Rev not %s not match expected %s", rev, expectedRev)
	}

	if err := v.Create(""); err != nil {
		t.Errorf("Error running create command with no revision: %s", err.Error())
	}
	if err = v.SetRev("testrev"); err != nil {
		t.Errorf("Error setting rev to testrev: %s", err.Error())
	}
}
