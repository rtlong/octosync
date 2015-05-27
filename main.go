package main

import (
	"fmt"
	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
	"gopkg.in/alecthomas/kingpin.v2"
	"log"
	"os"
	"path"
)

const githubTokenEnvVar = "GITHUB_TOKEN"

var (
	argDebug   = kingpin.Flag("debug", "Enable debug mode.").Bool()
	argOrgName = kingpin.Arg("orgName", "Name of github org to sync. Defaults to name of current working directory").String()
)

func orgName() string {
	if len(*argOrgName) > 0 {
		return *argOrgName
	}
	cwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	return path.Base(cwd)
}

func getGithubToken() (value string, err error) {
	value = os.Getenv(githubTokenEnvVar)
	if value != "" {
		return
	}
	return value, fmt.Errorf("You must set the %s variable with a Github access token", githubTokenEnvVar)
}

func buildClient(accessToken string) *github.Client {
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: accessToken},
	)
	httpClient := oauth2.NewClient(oauth2.NoContext, ts)
	return github.NewClient(httpClient)
}

type RepositoriesService interface {
	ListByOrg(string, *github.RepositoryListByOrgOptions) ([]github.Repository, *github.Response, error)
}

func getRepos(org string, repoService RepositoriesService) ([]github.Repository, *github.Response, error) {
	// TODO: Make this auto-fetch the next page and probably return repos through a channel
	return repoService.ListByOrg(org, &github.RepositoryListByOrgOptions{
		ListOptions: github.ListOptions{PerPage: 50},
	})
}

func repoIsCloned(name string) (isCloned bool, err error) {
	stat, err := os.Stat(path.Join(name, ".git"))
	if err == nil {
		if stat.IsDir() {
			isCloned = true
			return
		} else {
			return false, fmt.Errorf("Git-dir path exists but is not a directory: %s/.git", name)
		}
	}
	if os.IsNotExist(err) {
		stat, err = os.Stat(name)
		if err == nil {
			return false, fmt.Errorf("Clone directory already exists but is not a git repository: %s", name)
		}
		if os.IsNotExist(err) {
			return false, nil
		}
	}
	return
}

func handleError(err error, message string) {
	if err == nil {
		return
	}
	if message == "" {
		message = "Error: %s"
	}
	log.Fatalf(message, err)
}

func main() {
	kingpin.Version("0.0.1")
	kingpin.Parse()

	accessToken, err := getGithubToken()
	handleError(err, "")

	client := buildClient(accessToken)

	// list all repositories for the authenticated user
	org := orgName()
	log.Printf("Looking up repos for org '%s'", org)
	repos, _, err := getRepos(org, client.Repositories)
	handleError(err, "")

	for _, repo := range repos {
		log.Println(*repo.Name)
		isCloned, err := repoIsCloned(*repo.Name)
		if err != nil {
			log.Printf("  error: %s", err)
		} else if isCloned {
			log.Println("  would fetch")
		} else {
			log.Println("  would clone")
		}
	}
}
