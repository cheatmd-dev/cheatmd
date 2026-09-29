package convert

import "testing"

func TestConvertTldrGeneratedNamesStayUnique(t *testing.T) {
	for _, tc := range []struct {
		name, command, wantCommand, wantVars string
	}{
		{
			name:        "natural suffix after collision",
			command:     "echo {{a-b}} {{a_b}} {{a_b_2}} {{a-b}}",
			wantCommand: "echo $a_b $a_b_2 $a_b_2_2 $a_b",
			wantVars: "var a_b = printf '%s\\n' 'a-b' --- --header \"a-b\"\n" +
				"var a_b_2 = printf '%s\\n' 'a_b' --- --header \"a_b\"\n" +
				"var a_b_2_2 = printf '%s\\n' 'a_b_2' --- --header \"a_b_2\"\n",
		},
		{
			name:        "natural suffix before collision",
			command:     "echo {{a_b_2}} {{a_b}} {{a-b}} {{a_b_2}}",
			wantCommand: "echo $a_b_2 $a_b $a_b_3 $a_b_2",
			wantVars: "var a_b_2 = printf '%s\\n' 'a_b_2' --- --header \"a_b_2\"\n" +
				"var a_b = printf '%s\\n' 'a_b' --- --header \"a_b\"\n" +
				"var a_b_3 = printf '%s\\n' 'a-b' --- --header \"a-b\"\n",
		},
		{
			name:        "multiple reserved suffixes",
			command:     "echo {{a-b}} {{a_b_2}} {{a_b_3}} {{a_b}} {{a_b}}",
			wantCommand: "echo $a_b $a_b_2 $a_b_3 $a_b_4 $a_b_4",
			wantVars: "var a_b = printf '%s\\n' 'a-b' --- --header \"a-b\"\n" +
				"var a_b_2 = printf '%s\\n' 'a_b_2' --- --header \"a_b_2\"\n" +
				"var a_b_3 = printf '%s\\n' 'a_b_3' --- --header \"a_b_3\"\n" +
				"var a_b_4 = printf '%s\\n' 'a_b' --- --header \"a_b\"\n",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			input := "# echo\n- Print values:\n`" + tc.command + "`\n"
			got, err := ConvertTldr(input, "echo.md")
			if err != nil {
				t.Fatal(err)
			}
			want := "# Converted TLDR for Echo\n\n## Print values\n\n```sh\n" +
				tc.wantCommand + "\n```\n<!-- cheat\n" + tc.wantVars + "-->\n\n"
			if got != want {
				t.Fatalf("ConvertTldr() =\n%s\nwant:\n%s", got, want)
			}
		})
	}
}
