package crossref_test

import (
	"commonmeta/crossref"
	"commonmeta/doiutils"
	"commonmeta/types"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestGetCrossref(t *testing.T) {
	t.Parallel()

	type testCase struct {
		id   string
		want string
		err  error
	}

	journalArticle := types.Content{
		ID:        "https://doi.org/10.7554/elife.01567",
		Publisher: "eLife Sciences Publications, Ltd",
	}
	postedContent := types.Content{
		ID:        "https://doi.org/10.1101/097196",
		Publisher: "Cold Spring Harbor Laboratory",
	}

	testCases := []testCase{
		{id: journalArticle.ID, want: journalArticle.Publisher, err: nil},
		{id: postedContent.ID, want: postedContent.Publisher, err: nil},
	}
	for _, tc := range testCases {
		got, err := crossref.GetCrossref(tc.id)
		if tc.want != got.Publisher {
			t.Errorf("Get Crossref(%v): want %v, got %v, error %v",
				tc.id, tc.want, got, err)
		}
	}
}

func TestFetchCrossref(t *testing.T) {
	t.Parallel()
	type testCase struct {
		name string
		id   string
	}

	testCases := []testCase{
		{name: "test doi", id: "https://doi.org/10.5555/12345678"},
		{name: "journal article with data citation", id: "https://doi.org/10.7554/elife.01567"},
		{name: "posted content", id: "https://doi.org/10.1101/097196"},
		{name: "book", id: "https://doi.org/10.1017/9781108348843"},
		{name: "book chapter", id: "https://doi.org/10.1007/978-3-662-46370-3_13"},
		{name: "proceedings article", id: "https://doi.org/10.1145/3448016.3452841"},
		{name: "dataset", id: "https://doi.org/10.2210/pdb4hhb/pdb"},
		{name: "component", id: "https://doi.org/10.1371/journal.pmed.0030277.g001"},
		{name: "peer review", id: "https://doi.org/10.7554/elife.55167.sa2"},
		{name: "blog post", id: "https://doi.org/10.59350/2shz7-ehx26"},
		{name: "dissertation", id: "https://doi.org/10.14264/uql.2020.791"},
	}
	for _, tc := range testCases {
		got, err := crossref.FetchCrossref(tc.id)
		if err != nil {
			t.Errorf("Crossref Metadata(%v): error %v", tc.id, err)
		}
		// read json file from testdata folder and convert to Data struct
		doi, ok := doiutils.ValidateDOI(tc.id)
		if !ok {
			t.Fatal("invalid doi")
		}
		filename := strings.ReplaceAll(doi, "/", "_") + ".json"
		filepath := filepath.Join("testdata", filename)
		content, err := os.ReadFile(filepath)
		if err != nil {
			t.Fatal(err)
		}
		want := types.Data{}
		err = json.Unmarshal(content, &want)
		if err != nil {
			t.Fatal(err)
		}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("FetchCrossref(%s) mismatch (-want +got):\n%s", tc.id, diff)
		}
	}
}
