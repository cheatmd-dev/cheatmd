package headless

import (
	"bytes"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/cheatmd-dev/cheatmd/internal/resolver"
	"github.com/cheatmd-dev/cheatmd/pkg/config"
	"github.com/cheatmd-dev/cheatmd/pkg/parser"
)

func TestHeadlessSelectorResponse(t *testing.T) {
	previousShell := config.Get().Shell
	config.Get().Shell = "/bin/sh"
	t.Cleanup(func() { config.Get().Shell = previousShell })

	const columns = "--delimiter : --select-column 1"
	for _, tc := range []struct {
		name, args, rows, reply, want string
		expectsOptions                []string
	}{
		{"custom value", columns, "alpha:First\nbeta:Second", `"localhost:8080"`, "localhost:8080", []string{"alpha:First", "beta:Second"}},
		{"no transform", "", "alpha:First", `"alpha:First"`, "alpha:First", []string{"alpha:First"}},
		{"display column only", "--delimiter : --column 2", "alpha:First", `"First"`, "First", []string{"First"}},
		{"selected row", columns, "alpha:First\nbeta:Second", `"beta:Second"`, "beta", []string{"alpha:First", "beta:Second"}},
		{"duplicate identical rows", columns, "alpha:First\nalpha:First", `"alpha:First"`, "alpha", []string{"alpha:First", "alpha:First"}},
		{"ambiguous label", columns + " --column 2", "alpha:Same\nbeta:Same", `"Same"`, "Same", []string{"Same", "Same"}},
		{"displayed label", columns + " --column 2", "alpha:First\nbeta:Second", `"Second"`, "beta", []string{"First", "Second"}},
		{"no options", columns, "", `"localhost:8080"`, "localhost:8080", nil},
		{"omitted reply uses prefill", columns, "alpha:First", "", "alpha:First", []string{"alpha:First"}},
		{"map receives selected column", columns + ` --column 2 --map "tr a-z A-Z"`, "alpha:First", `"First"`, "ALPHA", []string{"First"}},
		{"custom value is mapped whole", columns + ` --map "tr a-z A-Z"`, "alpha:First", `"beta:Second"`, "BETA:SECOND", []string{"alpha:First"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			values := "{}"
			if tc.reply != "" {
				values = `{"endpoint":` + tc.reply + "}"
			}
			var output bytes.Buffer
			session := &RunnerSession{
				Vars: []resolver.VarState{{
					Def:     parser.VarDef{Name: "endpoint", Shell: "list-endpoints", Args: tc.args},
					Prefill: "alpha:First",
				}},
				Exec:    &mockHeadlessExecutor{shellResult: tc.rows},
				Decoder: json.NewDecoder(strings.NewReader(`{"result":{"values":` + values + `}}`)),
				Out:     &output,
			}
			if err := session.promptNextBatch(); err != nil {
				t.Fatal(err)
			}
			var request struct {
				Params struct {
					Variables []promptVar `json:"variables"`
				} `json:"params"`
			}
			if err := json.Unmarshal(output.Bytes(), &request); err != nil {
				t.Fatal(err)
			}
			if got := request.Params.Variables[0].Options; !reflect.DeepEqual(got, tc.expectsOptions) {
				t.Errorf("options = %q, want %q", got, tc.expectsOptions)
			}
			if v := session.Vars[0]; !v.Resolved || v.Value != tc.want {
				t.Errorf("resolved = %v, value = %q, want resolved value %q", v.Resolved, v.Value, tc.want)
			}
		})
	}
}
