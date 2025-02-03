package lib

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/shurcooL/githubv4"
	"golang.org/x/oauth2"
)

type GithubClient struct {
	client *githubv4.Client
	ctx    context.Context
}

type ProjectDetails struct {
	ID           string
	FieldsByID   map[string]interface{}
	FieldsByName map[string]interface{}

	// SingleSelectFields map[string]SingleSelectField
	// Fields             map[string]Node
}

type SingleSelectField struct {
	ID      string
	Name    string
	Options map[string]Field `json:"options,omitempty"`
}

type Field struct {
	ID   string
	Name string
}

func NewGithubClient() *GithubClient {
	src := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: os.Getenv("GITHUB_TOKEN")},
	)
	httpClient := oauth2.NewClient(context.Background(), src)

	client := githubv4.NewClient(httpClient)
	return &GithubClient{
		client: client,
		ctx:    context.Background(),
	}
}

type ProjectItem struct {
	Status    string
	StartDate string
	EndDate   string
}

func (g *GithubClient) UpdateProjectItem(projectID, itemID, fieldID string, value githubv4.ProjectV2FieldValue) error {
	var query struct {
		UpdateProjectV2ItemFieldValue struct {
			ProjectV2Item struct {
				ID githubv4.String
			} `graphql:"projectV2Item"`
		} `graphql:"updateProjectV2ItemFieldValue(input: $input)"`
	}

	fmt.Println("Updating project item")
	fmt.Println(projectID, itemID, fieldID, value.Date)
	input := githubv4.UpdateProjectV2ItemFieldValueInput{
		ProjectID: githubv4.ID(projectID),
		ItemID:    githubv4.ID(itemID),
		FieldID:   githubv4.ID(fieldID),
		Value:     value,
	}

	return g.client.Mutate(g.ctx, &query, input, nil)
}

func (g *GithubClient) FetchStatusAndStartDate(projectItemID string) (*ProjectItem, error) {
	var query struct {
		Node struct {
			ID            githubv4.String
			ProjectV2Item struct {
				CurrentStatus struct {
					ProjectV2ItemFieldSingleSelectValue struct {
						Name githubv4.String
					} `graphql:"... on ProjectV2ItemFieldSingleSelectValue"`
				} `graphql:"currentStatus: fieldValueByName(name: \"Status\")"`
				CurrentStartDate struct {
					ProjectV2ItemFieldDateValue struct {
						Date      githubv4.String
						UpdatedAt githubv4.Date
					} `graphql:"... on ProjectV2ItemFieldDateValue"`
				} `graphql:"currentStartDate: fieldValueByName(name: \"Start date\")"`
				CurrentEndDate struct {
					ProjectV2ItemFieldDateValue struct {
						Date      githubv4.String
						UpdatedAt githubv4.Date
					} `graphql:"... on ProjectV2ItemFieldDateValue"`
				} `graphql:"currentEndDate: fieldValueByName(name: \"End date\")"`
			} `graphql:"... on ProjectV2Item"`
		} `graphql:"node(id: $projectItemID)"`
	}
	variables := map[string]interface{}{
		"projectItemID": githubv4.ID(projectItemID),
	}
	err := g.client.Query(g.ctx, &query, variables)
	if err != nil {
		return nil, err
	}

	startDateStr := query.Node.ProjectV2Item.CurrentStartDate.ProjectV2ItemFieldDateValue.Date
	endDateStr := query.Node.ProjectV2Item.CurrentEndDate.ProjectV2ItemFieldDateValue.Date
	return &ProjectItem{
		Status:    string(query.Node.ProjectV2Item.CurrentStatus.ProjectV2ItemFieldSingleSelectValue.Name),
		StartDate: string(startDateStr),
		EndDate:   string(endDateStr),
	}, nil

}

func (g *GithubClient) ProjectDetails(organization string, projectNumber int) (*ProjectDetails, error) {
	var orgInfoQuery struct {
		Organization struct {
			ProjectV2 struct {
				ID     githubv4.String
				Fields struct {
					Nodes []struct {
						ProjectV2Field struct {
							ID   githubv4.String
							Name githubv4.String
						} `graphql:"... on ProjectV2Field"`
						ProjectV2SingleSelectField struct {
							ID      githubv4.String
							Name    githubv4.String
							Options []struct {
								ID   githubv4.String
								Name githubv4.String
							}
						} `graphql:"... on ProjectV2SingleSelectField"`
						ProjectV2IterationField struct {
							ID            githubv4.String
							Name          githubv4.String
							Configuration struct {
								Iterations []struct {
									ID        githubv4.String
									StartDate githubv4.String
								}
							}
						} `graphql:"... on ProjectV2IterationField"`
					} `graphql:"nodes"`
				} `graphql:"fields(first: 100)"`
			} `graphql:"projectV2(number: $projectNumber)"`
		} `graphql:"organization(login: $organization)"`
	}
	variables := map[string]interface{}{
		"organization":  githubv4.String(organization),
		"projectNumber": githubv4.Int(projectNumber),
	}
	err := g.client.Query(g.ctx, &orgInfoQuery, variables)
	if err != nil {
		return nil, err
	}

	projectDetails := &ProjectDetails{}
	projectDetails.ID = string(orgInfoQuery.Organization.ProjectV2.ID)
	projectDetails.FieldsByName = make(map[string]interface{})
	projectDetails.FieldsByID = make(map[string]interface{})

	for _, field := range orgInfoQuery.Organization.ProjectV2.Fields.Nodes {
		var fieldValue interface{}
		var fieldName, fieldID string

		if field.ProjectV2Field.Name != "" {
			fieldName = string(field.ProjectV2Field.Name)
			fieldID = string(field.ProjectV2Field.ID)
			fieldValue = Field{
				ID:   string(field.ProjectV2Field.ID),
				Name: string(field.ProjectV2Field.Name),
			}
		}

		if field.ProjectV2SingleSelectField.Name != "" {
			fieldName = string(field.ProjectV2SingleSelectField.Name)
			fieldID = string(field.ProjectV2SingleSelectField.ID)

			optionsMap := make(map[string]Field)
			for _, option := range field.ProjectV2SingleSelectField.Options {
				optionsMap[string(option.ID)] = Field{
					ID:   string(option.ID),
					Name: string(option.Name),
				}
			}

			fieldValue = SingleSelectField{
				ID:      string(field.ProjectV2SingleSelectField.ID),
				Name:    fieldName,
				Options: optionsMap,
			}
		}

		projectDetails.FieldsByName[fieldName] = fieldValue
		projectDetails.FieldsByID[fieldID] = fieldValue
	}
	return projectDetails, nil

}

func (g *GithubClient) FieldIDs(projectID string) (map[string]string, error) {
	var query struct {
		Node struct {
			ProjectV2 struct {
				Fields struct {
					Nodes []struct {
						ProjectV2FieldCommon struct {
							ID   githubv4.String
							Name githubv4.String
						} `graphql:"... on ProjectV2FieldCommon"`
					} `graphql:"nodes"`
				} `graphql:"fields(first: 100)"`
			} `graphql:"... on ProjectV2"`
		} `graphql:"node(id: $id)"`
	}
	variables := map[string]interface{}{
		"id": githubv4.ID(projectID),
	}
	err := g.client.Query(g.ctx, &query, variables)
	if err != nil {
		return nil, err
	}
	fieldIDs := make(map[string]string)
	for _, field := range query.Node.ProjectV2.Fields.Nodes {
		fieldIDs[strings.ToLower(string(field.ProjectV2FieldCommon.Name))] = string(field.ProjectV2FieldCommon.ID)
	}
	return fieldIDs, err
}
