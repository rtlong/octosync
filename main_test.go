package main

import (
	"fmt"
	"github.com/google/go-github/github"
	. "github.com/smartystreets/goconvey/convey"
	"io/ioutil"
	"os"
	"testing"
)

type stubRepositoryService struct {
	listByOrg func(string, *github.RepositoryListByOrgOptions) ([]github.Repository, *github.Response, error)
}

func (r *stubRepositoryService) ListByOrg(org string, opts *github.RepositoryListByOrgOptions) ([]github.Repository, *github.Response, error) {
	return r.listByOrg(org, opts)
}

func useTempDir() string {
	name, err := ioutil.TempDir("", "octosync")
	if err != nil {
		panic(err)
	}
	err = os.Chdir(name)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Using temp dir %s\n", name)
	return name
}

func cleanupTempDir(name string) {
	err := os.RemoveAll(name)
	if err != nil {
		panic(err)
	}
}

func TestMain(t *testing.T) {
	Convey("main", t, func() {
		Convey("orgName()", func() {
			Convey("without an any CLI argument", func() {
				*argOrgName = ""
				Convey("returns the basename of the $PWD", func() {
					So(orgName(), ShouldEqual, "octosync") // FIXME: kinda gross to rely on this hard-coded
				})
			})

			Convey("with an argument on the CLI", func() {
				*argOrgName = "foo"
				Convey("returns the argument", func() {
					So(orgName(), ShouldEqual, "foo")
				})
			})
		})

		Convey("getGithubToken()", func() {
			Convey("it pulls the GITHUB_TOKEN environment variable", func() {
				os.Setenv("GITHUB_TOKEN", "foobarbaz")
				value, err := getGithubToken()
				So(value, ShouldEqual, "foobarbaz")
				So(err, ShouldBeNil)

				os.Unsetenv("GITHUB_TOKEN")
				value, err = getGithubToken()
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "GITHUB_TOKEN")
			})
		})

		Convey("getRepos()", func() {
			var (
				retval struct {
					repos    []github.Repository
					response *github.Response
					err      error
				}
				callargs struct {
					org  string
					opts *github.RepositoryListByOrgOptions
				}
				id1, id2    int = 1, 2
				repoService     = &stubRepositoryService{
					listByOrg: func(org string, opts *github.RepositoryListByOrgOptions) ([]github.Repository, *github.Response, error) {
						callargs.org = org
						callargs.opts = opts
						return retval.repos, retval.response, retval.err
					},
				}
			)

			Convey("it proxies RepositoriesService.ListByOrg", func() {
				retval.repos = []github.Repository{
					{ID: &id1},
					{ID: &id2},
				}
				retval.err = fmt.Errorf("yo this is an error")
				retval.response = &github.Response{NextPage: 4, PrevPage: 2}

				repos, response, err := getRepos("myAwesomeOrg", repoService)

				So(repos, ShouldResemble, retval.repos)
				So(err, ShouldEqual, retval.err)
				So(response, ShouldEqual, retval.response)
				So(callargs.org, ShouldEqual, "myAwesomeOrg")
			})

			Convey("it asks for larger pages than the default options", func() {
				getRepos("", repoService)
				So(callargs.opts, ShouldResemble, &github.RepositoryListByOrgOptions{
					ListOptions: github.ListOptions{PerPage: 50},
				})
			})
		})

		Convey("repoIsCloned(repoName)", func() {
			tempdir := useTempDir()
			defer cleanupTempDir(tempdir)

			Convey("when #{repoName}/.git directory exists", func() {
				os.MkdirAll("myRepo/.git", os.ModeDir|0775)

				Convey("returns true and no error", func() {
					val, err := repoIsCloned("myRepo")
					So(val, ShouldBeTrue)
					So(err, ShouldBeNil)
				})
			})

			Convey("when #{repoName}/.git exists as a non-directory", func() {
				os.MkdirAll("myRepo/", os.ModeDir|0775)
				file, err := os.Create("myRepo/.git")
				handleError(err, "")
				file.Close()

				Convey("returns false and an error", func() {
					val, err := repoIsCloned("myRepo")
					So(val, ShouldBeFalse)
					So(err, ShouldNotBeNil)
					So(os.IsNotExist(err), ShouldBeFalse)
					So(err.Error(), ShouldContainSubstring, "myRepo/.git")
				})
			})

			Convey("when #{repoName} exists but doesn't contain .git/", func() {
				os.MkdirAll("myRepo/", os.ModeDir|0775)

				Convey("returns false and an error", func() {
					val, err := repoIsCloned("myRepo")
					So(val, ShouldBeFalse)
					So(err, ShouldNotBeNil)
					So(os.IsNotExist(err), ShouldBeFalse)
				})
			})

			Convey("else", func() {
				Convey("returns false and no error", func() {
					val, err := repoIsCloned("myRepo")
					So(val, ShouldBeFalse)
					So(err, ShouldBeNil)
				})
			})

		})
	})
}
