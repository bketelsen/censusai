// Copyright 2018, OpenCensus Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package main

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	os "os"
	"time"

	ocagent "contrib.go.opencensus.io/exporter/ocagent"
	"github.com/basvanbeek/ocsql"
	_ "github.com/mattn/go-sqlite3"
	"go.opencensus.io/plugin/ochttp"
	"go.opencensus.io/plugin/ochttp/propagation/tracecontext"
	"go.opencensus.io/trace"
)

func main() {
	// Register stats and trace exporters to export the collected data.
	serviceName := os.Getenv("SERVICE_NAME")
	if len(serviceName) == 0 {
		serviceName = "censusai"
	}
	fmt.Printf(serviceName)
	agentEndpoint := os.Getenv("OCAGENT_TRACE_EXPORTER_ENDPOINT")

	if len(agentEndpoint) == 0 {
		agentEndpoint = fmt.Sprintf("%s:%d", ocagent.DefaultAgentHost, ocagent.DefaultAgentPort)
	}

	fmt.Printf(agentEndpoint)
	exporter, err := ocagent.NewExporter(ocagent.WithInsecure(), ocagent.WithServiceName(serviceName), ocagent.WithAddress(agentEndpoint))

	if err != nil {
		log.Printf("Failed to create the agent exporter: %v", err)
	}

	driverName, err := ocsql.Register("sqlite3", ocsql.WithAllTraceOptions())
	if err != nil {
		log.Fatalf("Failed to register the ocsql driver: %v", err)
	}
	db, err := sql.Open(driverName, "resource.db")
	if err != nil {
		log.Fatalf("Failed to open the SQL database: %v", err)
	}
	defer func() {
		db.Close()
		// Wait to 4 seconds so that the traces can be exported
		waitTime := 4 * time.Second
		log.Printf("Waiting for %s seconds to ensure all traces are exported before exiting", waitTime)
		<-time.After(waitTime)
	}()

	ctx, span := trace.StartSpan(context.Background(), "Span Demo App")
	defer span.End()

	// And for the cleanup
	_, err = db.ExecContext(ctx, `DROP TABLE names`)
	if err != nil {
		log.Printf("Failed to drop table: %v\n", err)
	}
	_, err = db.ExecContext(ctx, `CREATE TABLE names(
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            first VARCHAR(256),
            last VARCHAR(256)
        )`)

	if err != nil {
		log.Fatalf("Failed to create table: %v", err)
	}
	rs, err := db.ExecContext(ctx, `INSERT INTO names(first, last) VALUES (?, ?)`, "Brian", "Ketelsen")
	if err != nil {
		log.Fatalf("Failed to insert values into tables: %v", err)
	}

	id, err := rs.LastInsertId()
	if err != nil {
		log.Fatalf("Failed to retrieve lastInserted ID: %v", err)
	}

	trace.RegisterExporter(exporter)

	trace.ApplyConfig(trace.Config{DefaultSampler: trace.AlwaysSample()})

	client := &http.Client{Transport: &ochttp.Transport{Propagation: &tracecontext.HTTPFormat{}}}

	http.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		fmt.Fprintf(w, "hello world")

		var jsonStr = []byte(`[ { "url": "http://blank.org", "arguments": [] } ]`)
		r, _ := http.NewRequest("POST", "http://blank.org", bytes.NewBuffer(jsonStr))
		r.Header.Set("Content-Type", "application/json")

		// Propagate the trace header info in the outgoing requests.
		r = r.WithContext(req.Context())
		resp, err := client.Do(r)
		if err != nil {
			log.Println(err)
		} else {
			// TODO: handle response
			resp.Body.Close()
		}
	})

	http.HandleFunc("/db", func(w http.ResponseWriter, req *http.Request) {
		fCtx, fSpan := trace.StartSpan(req.Context(), "db.find")
		row := db.QueryRowContext(fCtx, `SELECT * from names where id=?`, id) // closure from above
		fSpan.End()
		type name struct {
			Id          int
			First, Last string
		}
		n1 := new(name)
		if err := row.Scan(&n1.Id, &n1.First, &n1.Last); err != nil {
			log.Fatalf("Failed to fetch row: %v", err)
		}

		fmt.Fprintf(w, "hello %s", n1.First)
	})

	log.Fatal(http.ListenAndServe(":50030", &ochttp.Handler{Propagation: &tracecontext.HTTPFormat{}}))

}
