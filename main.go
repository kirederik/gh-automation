package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
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

type ChangesetItem map[string]interface{}
type Changeset map[string]ChangesetItem

type GithubIncomingEvent struct {
	Event   string       `json:"event"`
	Payload EventPayload `json:"payload"`
}

type EventPayload struct {
	Action        string        `json:"action"`
	ProjectV2Item ProjectV2Item `json:"projects_v2_item"`
	Changes       Changeset     `json:"changes"`
	Organization  GithubEntity  `json:"organization"`
	Sender        GithubEntity  `json:"sender"`
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

	w.Write([]byte("OK"))
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
