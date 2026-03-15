package obsidian

import (
	"context"
	"testing"

	"github.com/go-go-golems/go-go-goja/pkg/obsidiancli"
	"github.com/stretchr/testify/require"
)

type fakeRunner struct {
	t         *testing.T
	handlers  map[string]func(call obsidiancli.CallOptions) (obsidiancli.Result, error)
	callCount map[string]int
}

func newFakeRunner(t *testing.T) *fakeRunner {
	return &fakeRunner{
		t:         t,
		handlers:  map[string]func(call obsidiancli.CallOptions) (obsidiancli.Result, error){},
		callCount: map[string]int{},
	}
}

func (f *fakeRunner) Run(_ context.Context, spec obsidiancli.CommandSpec, call obsidiancli.CallOptions) (obsidiancli.Result, error) {
	f.callCount[spec.Name]++
	handler, ok := f.handlers[spec.Name]
	require.Truef(f.t, ok, "unexpected command %s", spec.Name)
	return handler(call)
}

func TestReadResolvesFriendlyReferenceAndCachesContent(t *testing.T) {
	runner := newFakeRunner(t)
	runner.handlers[obsidiancli.SpecFilesList.Name] = func(call obsidiancli.CallOptions) (obsidiancli.Result, error) {
		require.Equal(t, "md", call.Parameters["ext"])
		return obsidiancli.Result{
			Parsed: []string{
				"ZK/Claims/ZK - 2a0 - Systems thinking.md",
				"Inbox/Scratch.md",
			},
		}, nil
	}
	runner.handlers[obsidiancli.SpecFileRead.Name] = func(call obsidiancli.CallOptions) (obsidiancli.Result, error) {
		require.Equal(t, "ZK/Claims/ZK - 2a0 - Systems thinking.md", call.Parameters["path"])
		return obsidiancli.Result{
			Parsed: "# Systems Thinking\n\n#software\n",
			Stdout: "# Systems Thinking\n\n#software\n",
		}, nil
	}

	client := NewClient(Config{}, runner)

	content, err := client.Read(context.Background(), "ZK - 2a0 - Systems thinking")
	require.NoError(t, err)
	require.Equal(t, "# Systems Thinking\n\n#software\n", content)

	content, err = client.Read(context.Background(), "ZK - 2a0 - Systems thinking")
	require.NoError(t, err)
	require.Equal(t, "# Systems Thinking\n\n#software\n", content)
	require.Equal(t, 1, runner.callCount[obsidiancli.SpecFileRead.Name])
}

func TestNoteLoadsDerivedMetadata(t *testing.T) {
	runner := newFakeRunner(t)
	runner.handlers[obsidiancli.SpecFileRead.Name] = func(call obsidiancli.CallOptions) (obsidiancli.Result, error) {
		require.Equal(t, "ZK/Claims/note.md", call.Parameters["path"])
		content := "---\nstatus: draft\n---\n# Title\n\nSee [[Other Note]]\n\n- [ ] task\n#software\n"
		return obsidiancli.Result{Parsed: content, Stdout: content}, nil
	}

	client := NewClient(Config{}, runner)
	note, err := client.Note(context.Background(), "ZK/Claims/note.md")
	require.NoError(t, err)

	tags, err := note.Tags(context.Background())
	require.NoError(t, err)
	require.Equal(t, []string{"software"}, tags)

	links, err := note.Wikilinks(context.Background())
	require.NoError(t, err)
	require.Equal(t, []string{"Other Note"}, links)

	frontmatter, err := note.Frontmatter(context.Background())
	require.NoError(t, err)
	require.Equal(t, "draft", frontmatter["status"])
}

func TestQueryUsesNativeAndPostFilters(t *testing.T) {
	runner := newFakeRunner(t)
	runner.handlers[obsidiancli.SpecFilesList.Name] = func(call obsidiancli.CallOptions) (obsidiancli.Result, error) {
		require.Equal(t, "ZK/Claims", call.Parameters["folder"])
		require.Equal(t, "md", call.Parameters["ext"])
		return obsidiancli.Result{
			Parsed: []string{
				"ZK/Claims/Systems.md",
				"ZK/Claims/Architecture.md",
			},
		}, nil
	}
	runner.handlers[obsidiancli.SpecFileRead.Name] = func(call obsidiancli.CallOptions) (obsidiancli.Result, error) {
		path := call.Parameters["path"].(string)
		switch path {
		case "ZK/Claims/Systems.md":
			content := "# Systems\n\n#software\n"
			return obsidiancli.Result{Parsed: content, Stdout: content}, nil
		case "ZK/Claims/Architecture.md":
			content := "# Architecture\n\n#design\n"
			return obsidiancli.Result{Parsed: content, Stdout: content}, nil
		default:
			t.Fatalf("unexpected path: %s", path)
			return obsidiancli.Result{}, nil
		}
	}

	client := NewClient(Config{}, runner)
	results, err := client.Query().
		InFolder("ZK/Claims").
		WithExtension("md").
		NameContains("sys").
		Tagged("software").
		Run(context.Background())
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.Equal(t, "ZK/Claims/Systems.md", results[0].Path)
}

func TestBatchRunsSequentiallyAndPreservesPerNoteErrors(t *testing.T) {
	runner := newFakeRunner(t)
	runner.handlers[obsidiancli.SpecFilesList.Name] = func(call obsidiancli.CallOptions) (obsidiancli.Result, error) {
		return obsidiancli.Result{
			Parsed: []string{
				"ZK/Claims/One.md",
				"ZK/Claims/Two.md",
			},
		}, nil
	}

	client := NewClient(Config{}, runner)
	results, err := client.Batch(context.Background(), client.Query(), func(_ context.Context, note *Note) (any, error) {
		return note.Path, nil
	})
	require.NoError(t, err)
	require.Equal(t, []BatchItemResult{
		{Path: "ZK/Claims/One.md", Value: "ZK/Claims/One.md", Err: nil},
		{Path: "ZK/Claims/Two.md", Value: "ZK/Claims/Two.md", Err: nil},
	}, results)
}
