package engine

import (
	"fmt"
	"os/exec"
	"strings"
)

// ExportDiagram exports the pipeline dataflow to a diagram.
func (e *Engine) ExportDiagram(outfile string) {
	e.exportPipeline(outfile, "png", "dot", "72", "-Gsize=10,5 -Gdpi=200")
}

func (e *Engine) exportPipeline(outfile string, format string, layout string, scale string, more string) {
	dot := `digraph dbus {
    rankdir=LR
    node[width=1 fixedsize=true shape=circle style=filled fillcolor="darkorchid1" ]

    `

	lonelyInputs := make(map[string]struct{})
	lonelyFilters := make(map[string]struct{})

	for in := range e.InputRunners {
		lonelyInputs[in] = struct{}{}
	}

	// filter matchers
	for _, m := range e.router.filterMatchers {
		lonelyFilters[m.runner.Name()] = struct{}{}

		for source := range m.matches {
			link := fmt.Sprintf(`%s -> %s [label="Filter"]`, source, m.runner.Name())
			dot += "\r\n" + link

			delete(lonelyInputs, source)
		}
	}

	// output matchers
	for _, m := range e.router.outputMatchers {
		for source := range m.matches {
			link := fmt.Sprintf(`%s -> %s [label="Output"]`, source, m.runner.Name())
			dot += "\r\n" + link

			delete(lonelyFilters, source)
			delete(lonelyInputs, source)
		}
	}

	// the isolated plugins
	for p := range lonelyInputs {
		dot += "\r\n" + p
	}
	for p := range lonelyFilters {
		dot += "\r\n" + p
	}

	dot += "\r\n}"

	// the cmdLine is internal generated, should not vulnerable to security attack
	cmdLine := fmt.Sprintf("dot -o%s -T%s -K%s -s%s %s", outfile, format, layout, scale, more)
	cmd := exec.Command(`/bin/sh`, `-c`, cmdLine)
	cmd.Stdin = strings.NewReader(dot)

	cmd.Run()
}
