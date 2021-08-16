package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/chromedp/cdproto/emulation"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
	"github.com/friendsofgo/graphiql"
	"github.com/graphql-go/graphql"
)

//Job struct
type Job struct {
	ID             int      `json:"id"`
	Position       string   `json:"position"`
	Company        string   `json:"company"`
	Description    string   `json:"description"`
	SkillsRequired []string `json:"skillsRequired"`
	Location       string   `json:"location"`
	EmploymentType string   `json:"employmentType"`
}

var jobType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "Job",
		Fields: graphql.Fields{
			"id": &graphql.Field{
				Type: graphql.Int,
			},
			"position": &graphql.Field{
				Type: graphql.String,
			},
			"company": &graphql.Field{
				Type: graphql.String,
			},
			"description": &graphql.Field{
				Type: graphql.String,
			},
			"location": &graphql.Field{
				Type: graphql.String,
			},
			"employmentType": &graphql.Field{
				Type: graphql.String,
			},
			"skillsRequired": &graphql.Field{
				Type: graphql.NewList(graphql.String),
			},
		},
	},
)

func gqlHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Body == nil {
			http.Error(w, "No query data", 400)
			return
		}

		var rBody reqBody
		err := json.NewDecoder(r.Body).Decode(&rBody)
		if err != nil {
			http.Error(w, "Error parsing JSON request body", 400)
		}

		fmt.Fprintf(w, "%s", processQuery(rBody.Query))

	})
}

func processQuery(query string) (result string) {

	retrieveJobs := retrieveJobsFromFile()

	params := graphql.Params{Schema: gqlSchema(retrieveJobs), RequestString: query}
	r := graphql.Do(params)
	if len(r.Errors) > 0 {
		fmt.Printf("failed to execute graphql operation, errors: %+v", r.Errors)
	}
	rJSON, _ := json.Marshal(r)

	return fmt.Sprintf("%s", rJSON)

}

//Open the file data.json and retrieve json data
func retrieveJobsFromFile() func() []Job {
	return func() []Job {
		jsonf, err := os.Open("data.json")

		if err != nil {
			fmt.Printf("failed to open json file, error: %v", err)
		}

		jsonDataFromFile, _ := ioutil.ReadAll(jsonf)
		defer jsonf.Close()

		var jobsData []Job

		err = json.Unmarshal(jsonDataFromFile, &jobsData)

		if err != nil {
			fmt.Printf("failed to parse json, error: %v", err)
		}

		return jobsData
	}
}

func webserve() {
	graphiqlHandler, err := graphiql.NewGraphiqlHandler("/graphql")
	if err != nil {
		panic(err)
	}
	http.Handle("/graphql", gqlHandler())
	http.Handle("/graphiql", graphiqlHandler)
	http.ListenAndServe(":3000", nil)
}

type reqBody struct {
	Query string `json:"query"`
}

// Define the GraphQL Schema
func gqlSchema(queryJobs func() []Job) graphql.Schema {
	fields := graphql.Fields{
		"jobs": &graphql.Field{
			Type:        graphql.NewList(jobType),
			Description: "All Jobs",
			Resolve: func(params graphql.ResolveParams) (interface{}, error) {
				return queryJobs(), nil
			},
		},
		"job": &graphql.Field{
			Type:        jobType,
			Description: "Get Jobs by ID",
			Args: graphql.FieldConfigArgument{
				"id": &graphql.ArgumentConfig{
					Type: graphql.Int,
				},
			},
			Resolve: func(params graphql.ResolveParams) (interface{}, error) {
				id, success := params.Args["id"].(int)
				if success {
					for _, job := range queryJobs() {
						if int(job.ID) == id {
							return job, nil
						}
					}
				}
				return nil, nil
			},
		},
	}
	rootQuery := graphql.ObjectConfig{Name: "RootQuery", Fields: fields}
	schemaConfig := graphql.SchemaConfig{Query: graphql.NewObject(rootQuery)}
	schema, err := graphql.NewSchema(schemaConfig)
	if err != nil {
		fmt.Printf("failed to create new schema, error: %v", err)
	}

	return schema

}

func main() {
	// webserve()
	http.HandleFunc("/", myHandler)
	http.ListenAndServe(":9090", nil)

}

func pdfGrabber(url string, sel string, res *[]byte) chromedp.Tasks {
	// var res string

	start := time.Now()
	return chromedp.Tasks{
		emulation.SetUserAgentOverride("WebScraper 1.0"),
		chromedp.Navigate(url),
		// wait for footer element is visible (ie, page is loaded)
		// chromedp.ScrollIntoView(`footer`),
		chromedp.WaitVisible(`body`, chromedp.ByQuery),
		// chromedp.Text(`h1`, &res, chromedp.NodeVisible, chromedp.ByQuery),
		chromedp.ActionFunc(func(ctx context.Context) error {
			buf, _, err := page.PrintToPDF().WithPrintBackground(true).Do(ctx)
			if err != nil {
				return err
			}
			*res = buf
			//fmt.Printf("h1 contains: '%s'\n", res)
			fmt.Printf("\nTook: %f secs\n", time.Since(start).Seconds())
			return nil
		}),
	}
}

func myHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		// Form submitted
		r.ParseForm() // Required if you don't call r.FormValue()
		fmt.Println(r.PostForm["new_data"][0])
		fmt.Println("Scraping url now...")
		taskCtx, cancel := chromedp.NewContext(
			context.Background(),
			chromedp.WithLogf(log.Printf),
		)
		defer cancel()
		var pdfBuffer []byte
		if err := chromedp.Run(taskCtx, pdfGrabber(r.PostForm["new_data"][0], "body", &pdfBuffer)); err != nil {
			log.Fatal(err)
		}
		if err := ioutil.WriteFile("prescription.pdf", pdfBuffer, 0644); err != nil {
			log.Fatal(err)
		}
	}
	w.Write([]byte(dicky))
}

const htmlform = `
<html><body style="font-family: monospace">
<h3>select example to turn to pdf</h3>
<form action="process" method="post">
    <select id="new_data" name="new_data" class="tag-select chzn-done" multiple="" >
        <option value="%s">github.com</option>
        <option value="%s">wikipedia.org</option>
    </select>
    <input type="Submit" value="Send" />
</form>
</body></html>
`

var dicky = fmt.Sprintf(htmlform, "https://www.github.com", "https://www.wikipedia.org")
