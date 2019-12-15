package tasks

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	gitlab "github.com/xanzy/go-gitlab"
)

// MergeRequests task will send a Mattermost reminder for not removed branches.
func MergeRequests(gitlabURI, gitlabKey, gitlabProjectID, mattermostWebhookURI string, branchesToExlude []string, dryRun bool) error {
	mrs, err := getMergeRequests(gitlabURI, gitlabKey, gitlabProjectID, branchesToExlude)
	if err != nil {
		return err
	}

	return sendAll(mattermostWebhookURI, mrs, dryRun)
}

type mergeRequest struct {
	sourceBranch, targetBranch *gitlab.Branch
	event                      *gitlab.ContributionEvent
	mergeRequest               *gitlab.MergeRequest
}

func getMergeRequests(gitlabURI, gitlabKey, gitlabProjectID string,
	branchesToExclude []string) ([]mergeRequest, error) {
	mBranchesToExclude := map[string]string{}
	for _, b := range branchesToExclude {
		mBranchesToExclude[b] = b
	}

	git := gitlab.NewClient(nil, gitlabKey)

	if err := git.SetBaseURL(fmt.Sprintf("%s/api/v4", gitlabURI)); err != nil {
		return nil, errors.WithStack(err)
	}

	// get all merged and closed merge requests
	mergedMRs, err := getAllMergeRequests(git, gitlabProjectID, "merged")
	if err != nil {
		return nil, errors.WithStack(err)
	}
	closedMRs, err := getAllMergeRequests(git, gitlabProjectID, "closed")
	if err != nil {
		return nil, errors.WithStack(err)
	}
	mrs := append(mergedMRs, closedMRs...)
	logrus.Infof("Found %d merge requests closed or merged", len(mrs))

	// get project's branches
	bs, err := getAllBranches(git, gitlabProjectID)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	logrus.Infof("Found %d branches", len(bs))

	// get merge and close events
	mergeEs, err := getAllEvents(git, gitlabProjectID, gitlab.MergedEventType)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	closeEs, err := getAllEvents(git, gitlabProjectID, gitlab.ClosedEventType)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	es := append(mergeEs, closeEs...)
	logrus.Infof("Found %d merge events", len(es))

	// init map of branches
	mbs := map[string]*gitlab.Branch{}
	for _, b := range bs {
		mbs[b.Name] = b
	}

	// filter merge request with existing branches
	filtered := []*gitlab.MergeRequest{}
	for _, mr := range mrs {
		if _, ok := mbs[mr.SourceBranch]; ok {
			if _, ok := mBranchesToExclude[mr.SourceBranch]; !ok {
				filtered = append(filtered, mr)
			}
		}
	}
	logrus.Infof("Found %d merge requests with existing branches", len(filtered))

	// init map of merge event for merge request
	mes := map[int]*gitlab.ContributionEvent{}
	for _, e := range es {
		mes[e.TargetIID] = e
	}

	// show branches to removes
	results := []mergeRequest{}
	for _, mr := range filtered {
		results = append(results, mergeRequest{
			sourceBranch: mbs[mr.SourceBranch],
			targetBranch: mbs[mr.TargetBranch],
			event:        mes[mr.IID],
			mergeRequest: mr,
		})
	}

	return results, nil
}

func getAllMergeRequests(gitlabCLient *gitlab.Client, gitlabProjectID, state string) ([]*gitlab.MergeRequest, error) {
	results := []*gitlab.MergeRequest{}
	return results, getAllPages(func(currentPage int) (*gitlab.Response, error) {
		mrs, res, err := gitlabCLient.MergeRequests.ListProjectMergeRequests(gitlabProjectID,
			&gitlab.ListProjectMergeRequestsOptions{
				ListOptions: gitlab.ListOptions{Page: currentPage, PerPage: 100},
				State:       gitlab.String(state),
			})
		if err != nil {
			return nil, errors.WithStack(err)
		}

		results = append(results, mrs...)

		return res, nil
	})
}

func getAllEvents(gitlabClient *gitlab.Client, gitlabProjectID string, action gitlab.EventTypeValue) ([]*gitlab.ContributionEvent, error) {
	results := []*gitlab.ContributionEvent{}
	return results, getAllPages(func(currentPage int) (*gitlab.Response, error) {
		ces, res, err := gitlabClient.Events.ListProjectContributionEvents(gitlabProjectID,
			&gitlab.ListContributionEventsOptions{
				ListOptions: gitlab.ListOptions{Page: currentPage, PerPage: 100},
			}, func(req *http.Request) error {
				// fix for gitlab client
				req.URL.Opaque = strings.Replace(req.URL.Opaque, gitlabProjectID, fmt.Sprintf("projects/%s", gitlabProjectID), -1)
				q := req.URL.Query()
				q.Set("action", string(action))
				q.Set("target_type", string(gitlab.MergeRequestEventTargetType))
				req.URL.RawQuery = q.Encode()
				return nil
			})
		if err != nil {
			return nil, errors.WithStack(err)
		}

		results = append(results, ces...)

		return res, nil
	})
}

func getAllBranches(gitlabCLient *gitlab.Client, gitlabProjectID string) ([]*gitlab.Branch, error) {
	results := []*gitlab.Branch{}
	return results, getAllPages(func(currentPage int) (*gitlab.Response, error) {
		bs, res, err := gitlabCLient.Branches.ListBranches(gitlabProjectID,
			&gitlab.ListBranchesOptions{Page: currentPage, PerPage: 100})
		if err != nil {
			return nil, errors.WithStack(err)
		}

		results = append(results, bs...)

		return res, nil
	})
}

func getAllPages(f func(int) (*gitlab.Response, error)) error {
	for currentPage, totalPages := 1, 1; currentPage <= totalPages; currentPage++ {
		res, err := f(currentPage)
		if err != nil {
			return err
		}

		xTotalPages := res.Header.Get("X-Total-Pages")
		if xTotalPages != "" {
			totalPages, err = strconv.Atoi(xTotalPages)
			if err != nil {
				return errors.WithStack(err)
			}
		}

		logrus.Debugf("Get all pages (page %d/%d)", currentPage, totalPages)
	}
	return nil
}

type message struct {
	Text string `json:"text"`
}

func sendAll(mattermostWebhookURL string, mrs []mergeRequest, dryRun bool) error {
	if len(mrs) < 1 {
		return nil
	}

	mergedMRs := []mergeRequest{}
	for _, mr := range mrs {
		if mr.mergeRequest.State == "merged" {
			mergedMRs = append(mergedMRs, mr)
		}
	}

	mes := message{Text: fmt.Sprintf("Hi there are %d branche(s) to remove !", len(mergedMRs))}
	for _, mr := range mergedMRs {
		mes.Text = fmt.Sprintf("%sMR **[%d](%s)** was **%s** by @%s in **%s** and source branch **%s** still exists, please remove it.\n",
			mes.Text, mr.mergeRequest.IID, mr.mergeRequest.WebURL, mr.mergeRequest.State,
			mr.event.Author.Username, mr.targetBranch.Name, mr.sourceBranch.Name)
	}

	if !dryRun {
		data, err := json.Marshal(mes)
		if err != nil {
			return errors.WithStack(err)
		}

		if _, err := http.Post(mattermostWebhookURL, "application/json", bytes.NewReader(data)); err != nil {
			return errors.WithStack(err)
		}
	} else {
		logrus.Info(mes)
	}

	return nil
}
