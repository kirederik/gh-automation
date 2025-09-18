package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"slices"
	"time"

	"github.com/gorilla/mux"
	"github.com/kirederik/ghproject/lib"
	"github.com/shurcooL/githubv4"
)

type ProjectInfo struct {
	ID            string
	KanbanColumns []Node
}

type Node struct {
	Id   string
	Name string
}

type GithubEntity struct {
	Name string `json:"login"`
}

type ProjectV2Item struct {
	ID            int64        `json:"id"`
	NodeID        string       `json:"node_id"`
	ProjectNodeID string       `json:"project_node_id"`
	Creator       GithubEntity `json:"creator"`
	CreatedAt     string       `json:"created_at"`
	UpdatedAt     string       `json:"updated_at"`
	ArchivedAt    string       `json:"archived_at"`
}

type PullRequest struct {
	ID     int64  `json:"id"`
	NodeID string `json:"node_id"`
	Number int64  `json:"number"`
	State  string `json:"state"`
}

type Issue struct {
	ID     int64  `json:"id"`
	NodeID string `json:"node_id"`
	Number int64  `json:"number"`
	State  string `json:"state"`
	Title  string `json:"title"`
}

type Repository struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	FullName string `json:"full_name"`
}

type ChangesetItem map[string]interface{}
type Changeset map[string]ChangesetItem

type GithubIncomingEvent struct {
	Event   string       `json:"event"`
	Payload EventPayload `json:"payload"`
}
type EventPayload struct {
	Action        string         `json:"action"`
	ProjectV2Item *ProjectV2Item `json:"projects_v2_item,omitempty"`
	Changes       Changeset      `json:"changes"`
	Organization  GithubEntity   `json:"organization"`
	Sender        GithubEntity   `json:"sender"`
	PullRequest   *PullRequest   `json:"pull_request,omitempty"`
	Issue         *Issue         `json:"issue,omitempty"`
	Repository    *Repository    `json:"repository,omitempty"`
}

const (
	ReorderAction       = "reorder"
	EditedAction        = "edited"
	ReorderChangesetKey = "previous_projects_v2_item_node_id"
)

var (
	projectDetails *lib.ProjectDetails
	ghClient       *lib.GithubClient
)

func IncomingRequestHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Incoming request")
	body, err := io.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		http.Error(w, "Error reading request body", http.StatusInternalServerError)
		return
	}

	var event EventPayload
	if err := json.Unmarshal(body, &event); err != nil {
		fmt.Println(err)
		http.Error(w, "Error parsing request body", http.StatusBadRequest)
		return
	}

	fmt.Println("Request body: ", string(body[:100]))
	fmt.Println("Event action: ", event.Action)

	if event.ProjectV2Item != nil {
		handleProjectV2Item(event)
	}
	if event.PullRequest != nil {
		handlePullRequest(event)
	}
	if event.Issue != nil {
		handleIssue(event)
	}

	w.Write([]byte("OK"))
}

func handleIssue(event EventPayload) {
	fmt.Printf("Issue event: %s, issue %s#%d\n", event.Action, event.Repository.FullName, event.Issue.Number)
	projectID := projectDetails.ID
	if slices.Contains([]string{"edited", "reopened", "opened", "created"}, event.Action) {
		fmt.Printf("Adding issue %s#%d to project\n", event.Repository.FullName, event.Issue.Number)
		itemID, err := ghClient.AddNodeToProject(projectID, event.Issue.NodeID)
		if err != nil {
			log.Printf("Failed to add issue to project: %v", err)
			return
		}
		fmt.Printf("Added issue to project as item: %s\n", itemID)

		assignTypeToIssue(event.Issue.Title, event.Issue.NodeID)
	}
}

func assignTypeToIssue(title, issueNodeID string) {
	fmt.Printf("Attempting to assign type to issue with title: %q\n", title)

	typeName, found := projectDetails.TypeMapping.GetTypeFromTitle(title)
	if !found {
		fmt.Printf("No matching type found for title: %s\n", title)
		return
	}

	fmt.Printf("Detected type: %s\n", typeName)

	issueTypeID, exists := projectDetails.TypeMapping.GetTypeID(typeName)
	if !exists {
		fmt.Printf("Type '%s' not found in organization issue types\n", typeName)
		return
	}

	// Update the issue with the detected type
	err := ghClient.UpdateIssueType(issueNodeID, issueTypeID)
	if err != nil {
		log.Printf("Failed to update issue type: %v", err)
		return
	}

	fmt.Printf("Successfully assigned type '%s' to issue\n", typeName)
}

func handlePullRequest(event EventPayload) {
	fmt.Printf("Pull request event: %s, PR %s#%d\n", event.Action, event.Repository.FullName, event.PullRequest.Number)
	projectID := projectDetails.ID
	if event.Action == "opened" {
		fmt.Printf("Adding PR %s#%d to project\n", event.Repository.FullName, event.PullRequest.Number)
		itemID, err := ghClient.AddNodeToProject(projectID, event.PullRequest.NodeID)
		if err != nil {
			log.Printf("Failed to add PR to project: %v", err)
			return
		}
		fmt.Printf("Added PR to project as item: %s\n", itemID)
	}
}

func handleProjectV2Item(event EventPayload) {
	switch event.Action {
	case EditedAction:
		fmt.Println("Project item edited")
		fieldChanged, ok := event.Changes["field_value"]
		fmt.Println(fieldChanged)
		if !ok {
			fmt.Println("No field value change")
			break
		}

		fieldNodeID := fieldChanged["field_node_id"].(string)
		fieldType := fieldChanged["field_type"]

		switch fieldType {
		case "single_select":
			nodeUpdated := projectDetails.FieldsByID[fieldNodeID].(lib.SingleSelectField)
			fmt.Println("Field updated: ", nodeUpdated.Name)

			switch nodeUpdated.Name {
			case "Status":
				if event.ProjectV2Item.NodeID == "" {
					fmt.Println("No project item node ID")
					break
				}
				itemDetails, err := ghClient.FetchStatusAndStartDate(event.ProjectV2Item.NodeID)
				if err != nil {
					log.Println("Error here", err)
					return
				}

				var toUpdate string
				var value *githubv4.ProjectV2FieldValue

				value = &githubv4.ProjectV2FieldValue{
					Date: githubv4.NewDate(githubv4.Date{
						Time: time.Now(),
					}),
				}

				if itemDetails.Status == "In progress" && itemDetails.StartDate == "" {
					toUpdate = "Start date"
				}

				if itemDetails.Status == "Done" && itemDetails.EndDate == "" {
					toUpdate = "End date"
				}

				if toUpdate != "" {
					fmt.Println("Updating " + toUpdate)
					f := projectDetails.FieldsByName[toUpdate].(lib.SingleSelectField)
					err = ghClient.UpdateProjectItem(
						event.ProjectV2Item.ProjectNodeID,
						event.ProjectV2Item.NodeID,
						f.ID,
						*value,
					)
					if err != nil {
						log.Println(err)
					}
				}
			}
		}
	}
}

func main() {
	fmt.Println()
	fmt.Println("--- Starting the application ---")
	ghClient = lib.NewGithubClient()
	var err error
	projectDetails, err = ghClient.ProjectDetails("syntasso", 4)
	if err != nil {
		log.Fatal("error to query project details", err)
	}
	fmt.Println("Project ID: ", projectDetails.ID)

	r := mux.NewRouter()
	r.HandleFunc("/", IncomingRequestHandler).Methods("POST")

	srv := &http.Server{
		Addr:    ":8080",
		Handler: r,
	}

	log.Println("Server started on port 8080")

	wait := make(chan struct{})

	go func() {
		log.Fatal(srv.ListenAndServe())
	}()

	// go func() {
	// 	time.Sleep(time.Second / 2)
	// 	// call a local binary
	// 	cmd := exec.Command("./send.sh")
	// 	err := cmd.Run()
	// 	if err != nil {
	// 		log.Fatal(err)
	// 	}
	// }()

	<-wait
}
