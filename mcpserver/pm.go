// Package mcpserver exposes the daemon's control surface as MCP tools.
package mcpserver

import (
	"context"
	"encoding/json"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"morphstudio/control"
	"morphstudio/queue"
)

// ServerName is the name every morphd MCP server announces.
const ServerName = "morphd"

// ServerVersion is the version every morphd MCP server announces.
const ServerVersion = "0.1.0"

// Empty is the argument of a tool that takes no input.
type Empty struct{}

// jsonResult renders v as the one text content of a non-error tool result.
func jsonResult(v any) (*mcp.CallToolResult, any, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return errorResult(err)
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: string(b)}},
	}, nil, nil
}

// errorResult renders a Control error as the one text content of an error result.
func errorResult(err error) (*mcp.CallToolResult, any, error) {
	_, code := control.Code(err)
	return &mcp.CallToolResult{
		IsError: true,
		Content: []mcp.Content{&mcp.TextContent{Text: code + ": " + err.Error()}},
	}, nil, nil
}

// projectsView wraps the project list so a nil slice renders as [].
type projectsView struct {
	Projects []control.ProjectView `json:"projects"`
}

// okView is the {"ok":true} acknowledgement of a command tool.
type okView struct {
	OK bool `json:"ok"`
}

// questionView wraps a pending question so a nil question renders as null.
type questionView struct {
	Question *control.Question `json:"question"`
}

// stopView wraps a recorded stop so a nil stop renders as null.
type stopView struct {
	Stop *control.Stop `json:"stop"`
}

// projectCreateInput is the input of project_create.
type projectCreateInput struct {
	Name     string `json:"name" jsonschema:"the project name"`
	Language string `json:"language" jsonschema:"the project language"`
	RepoURL  string `json:"repo_url" jsonschema:"the git repository URL"`
}

// orderInput is the input of order.
type orderInput struct {
	Text string `json:"text" jsonschema:"the order to hand to the running session"`
}

// answerInput is the input of answer.
type answerInput struct {
	Option int `json:"option" jsonschema:"the index of the chosen option"`
}

// restartInput is the input of restart.
type restartInput struct {
	Force bool `json:"force,omitempty" jsonschema:"restart even when the session is busy"`
}

// planLoadInput is the input of plan_load.
type planLoadInput struct {
	Approved string           `json:"approved" jsonschema:"the hash of the approved plan"`
	Phases   []planPhaseInput `json:"phases" jsonschema:"the phases of the plan"`
}

// planPhaseInput is one phase of a plan_load call.
type planPhaseInput struct {
	ID        string     `json:"id" jsonschema:"the phase id"`
	StopAfter string     `json:"stop_after,omitempty" jsonschema:"the stop mark set after the phase"`
	Caps      *capsInput `json:"caps,omitempty" jsonschema:"the phase spend caps"`
}

// capsInput is the spend cap of one plan phase.
type capsInput struct {
	ClaudeUSD   float64 `json:"claude_usd,omitempty" jsonschema:"the Claude budget in US dollars"`
	Hours       float64 `json:"hours,omitempty" jsonschema:"the wall clock budget in hours"`
	ExecutorUSD float64 `json:"executor_usd,omitempty" jsonschema:"the executor budget in US dollars"`
}

// UserServer builds the user-level MCP server: the project list and creation.
func UserServer(c control.Control) *mcp.Server {
	s := mcp.NewServer(&mcp.Implementation{Name: ServerName, Version: ServerVersion}, nil)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "projects_list",
		Description: "List every project the daemon knows about.",
	}, func(_ context.Context, _ *mcp.CallToolRequest, _ Empty) (*mcp.CallToolResult, any, error) {
		views := c.Projects()
		if views == nil {
			views = []control.ProjectView{}
		}
		return jsonResult(projectsView{Projects: views})
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "project_create",
		Description: "Create a project from a name, a language and a repository URL.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in projectCreateInput) (*mcp.CallToolResult, any, error) {
		v, err := c.CreateProject(ctx, in.Name, in.Language, in.RepoURL)
		if err != nil {
			return errorResult(err)
		}
		return jsonResult(v)
	})

	return s
}

// PMServer builds the per-project MCP server around one project's control.
func PMServer(c control.Control, project string) *mcp.Server {
	s := mcp.NewServer(&mcp.Implementation{Name: ServerName, Version: ServerVersion}, nil)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "status",
		Description: "Report the project state, spend and queue position.",
	}, func(_ context.Context, _ *mcp.CallToolRequest, _ Empty) (*mcp.CallToolResult, any, error) {
		v, err := c.Status(project)
		if err != nil {
			return errorResult(err)
		}
		return jsonResult(v)
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "order",
		Description: "Hand an operator order to the running session.",
	}, func(_ context.Context, _ *mcp.CallToolRequest, in orderInput) (*mcp.CallToolResult, any, error) {
		v, err := c.Order(project, in.Text)
		if err != nil {
			return errorResult(err)
		}
		return jsonResult(v)
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "pending",
		Description: "Return the question the session is waiting on, if any.",
	}, func(_ context.Context, _ *mcp.CallToolRequest, _ Empty) (*mcp.CallToolResult, any, error) {
		q, err := c.Pending(project)
		if err != nil {
			return errorResult(err)
		}
		return jsonResult(questionView{Question: q})
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "answer",
		Description: "Answer the pending question with one of its options.",
	}, func(_ context.Context, _ *mcp.CallToolRequest, in answerInput) (*mcp.CallToolResult, any, error) {
		if err := c.Answer(project, in.Option); err != nil {
			return errorResult(err)
		}
		return jsonResult(okView{OK: true})
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "interrupt",
		Description: "Interrupt the running session.",
	}, func(_ context.Context, _ *mcp.CallToolRequest, _ Empty) (*mcp.CallToolResult, any, error) {
		if err := c.Interrupt(project); err != nil {
			return errorResult(err)
		}
		return jsonResult(okView{OK: true})
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "usage",
		Description: "Report the Claude limits and the session spend.",
	}, func(_ context.Context, _ *mcp.CallToolRequest, _ Empty) (*mcp.CallToolResult, any, error) {
		v, err := c.Usage(project)
		if err != nil {
			return errorResult(err)
		}
		return jsonResult(v)
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "restart",
		Description: "Restart the session, optionally forcing it.",
	}, func(_ context.Context, _ *mcp.CallToolRequest, in restartInput) (*mcp.CallToolResult, any, error) {
		if err := c.Restart(project, in.Force); err != nil {
			return errorResult(err)
		}
		return jsonResult(okView{OK: true})
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "plan_load",
		Description: "Load the approved plan and its phases into the queue.",
	}, func(_ context.Context, _ *mcp.CallToolRequest, in planLoadInput) (*mcp.CallToolResult, any, error) {
		plan := queue.Plan{Approved: in.Approved}
		for _, p := range in.Phases {
			phase := queue.Phase{ID: p.ID, StopAfter: p.StopAfter}
			if p.Caps != nil {
				phase.Caps = queue.Caps{
					ClaudeUSD:   p.Caps.ClaudeUSD,
					Hours:       p.Caps.Hours,
					ExecutorUSD: p.Caps.ExecutorUSD,
				}
			}
			plan.Phases = append(plan.Phases, phase)
		}
		v, err := c.PlanLoad(project, plan)
		if err != nil {
			return errorResult(err)
		}
		return jsonResult(v)
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "continue",
		Description: "Clear the current stop and resume the walk.",
	}, func(_ context.Context, _ *mcp.CallToolRequest, _ Empty) (*mcp.CallToolResult, any, error) {
		if err := c.Continue(project); err != nil {
			return errorResult(err)
		}
		return jsonResult(okView{OK: true})
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "stop_check",
		Description: "Return the stop the daemon recorded, if any.",
	}, func(_ context.Context, _ *mcp.CallToolRequest, _ Empty) (*mcp.CallToolResult, any, error) {
		v, err := c.StopCheck(project)
		if err != nil {
			return errorResult(err)
		}
		return jsonResult(stopView{Stop: v})
	})

	return s
}
