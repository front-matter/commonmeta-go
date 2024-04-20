package crossref

import (
	"commonmeta/dateutils"
	"commonmeta/doiutils"
	"commonmeta/types"
	"commonmeta/utils"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"reflect"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"
)

type Content struct {
	ID       string   `json:"id"`
	Type     string   `json:"type"`
	Abstract string   `json:"abstract"`
	Archive  []string `json:"archive"`
	Author   []struct {
		Given       string `json:"given"`
		Family      string `json:"family"`
		Name        string `json:"name"`
		ORCID       string `json:"ORCID"`
		Sequence    string `json:"sequence"`
		Affiliation []struct {
			ID []struct {
				ID     string `json:"id"`
				IDType string `json:"id-type"`
			} `json:"id"`
			Name string `json:"name"`
		} `json:"affiliation"`
	} `json:"author"`
	ContainerTitle []string   `json:"container-title"`
	DOI            string     `json:"doi"`
	Files          []struct{} `json:"files"`
	Funder         []struct {
		DOI   string   `json:"DOI"`
		Name  string   `json:"name"`
		Award []string `json:"award"`
	} `json:"funder"`
	GroupTitle string `json:"group-title"`
	Issue      string `json:"issue"`
	Published  struct {
		DateAsParts [][]int `json:"date-parts"`
		DateTime    string  `json:"date-time"`
	} `json:"published"`
	Issued struct {
		DateAsParts [][]int `json:"date-parts"`
		DateTime    string  `json:"date-time"`
	} `json:"issued"`
	Created struct {
		DateAsParts [][]int `json:"date-parts"`
		DateTime    string  `json:"date-time"`
	} `json:"created"`
	ISSN     []string `json:"ISSN"`
	ISBNType []struct {
		Value string `json:"value"`
		Type  string `json:"type"`
	} `json:"isbn-type"`
	Language string `json:"language"`
	License  []struct {
		Url            string `json:"URL"`
		ContentVersion string `json:"content-version"`
	} `json:"license"`
	Link []struct {
		ContentType string `json:"content-type"`
		Url         string `json:"url"`
	} `json:"link"`
	OriginalTitle []string `json:"original-title"`
	Page          string   `json:"page"`
	PublishedAt   string   `json:"published_at"`
	Publisher     string   `json:"publisher"`
	Reference     []struct {
		Key          string `json:"key"`
		DOI          string `json:"DOI"`
		ArticleTitle string `json:"article-title"`
		Year         string `json:"year"`
		Unstructured string `json:"unstructured"`
	} `json:"reference"`
	Relation struct {
		IsNewVersionOf []struct {
			ID     string `json:"id"`
			IDType string `json:"id-type"`
		} `json:"is-new-version-of"`
		IsPreviousVersionOf []struct {
			ID     string `json:"id"`
			IDType string `json:"id-type"`
		} `json:"is-previous-version-of"`
		IsVersionOf []struct {
			ID     string `json:"id"`
			IDType string `json:"id-type"`
		} `json:"is-version-of"`
		HasVersion []struct {
			ID     string `json:"id"`
			IDType string `json:"id-type"`
		} `json:"has-version"`
		IsPartOf []struct {
			ID     string `json:"id"`
			IDType string `json:"id-type"`
		} `json:"is-part-of"`
		HasPart []struct {
			ID     string `json:"id"`
			IDType string `json:"id-type"`
		} `json:"has-part"`
		IsVariantFormOf []struct {
			ID     string `json:"id"`
			IDType string `json:"id-type"`
		} `json:"is-variant-form-of"`
		IsOriginalFormOf []struct {
			ID     string `json:"id"`
			IDType string `json:"id-type"`
		} `json:"is-original-form-of"`
		IsIdenticalTo []struct {
			ID     string `json:"id"`
			IDType string `json:"id-type"`
		} `json:"is-identical-to"`
		IsTranslationOf []struct {
			ID     string `json:"id"`
			IDType string `json:"id-type"`
		} `json:"is-translation-of"`
		IsReviewedBy []struct {
			ID     string `json:"id"`
			IDType string `json:"id-type"`
		} `json:"is-reviewed-by"`
		Reviews []struct {
			ID     string `json:"id"`
			IDType string `json:"id-type"`
		} `json:"reviews"`
		HasReview []struct {
			ID     string `json:"id"`
			IDType string `json:"id-type"`
		} `json:"has-review"`
		IsPreprintOf []struct {
			ID     string `json:"id"`
			IDType string `json:"id-type"`
		} `json:"is-preprint-of"`
		HasPreprint []struct {
			ID     string `json:"id"`
			IDType string `json:"id-type"`
		} `json:"has-preprint"`
		IsSupplementTo []struct {
			ID     string `json:"id"`
			IDType string `json:"id-type"`
		} `json:"is-supplement-to"`
		IsSupplementedBy []struct {
			ID     string `json:"id"`
			IDType string `json:"id-type"`
		} `json:"is-supplemented-by"`
	} `json:"relation"`
	Resource struct {
		Primary struct {
			ContentType string `json:"content_type"`
			URL         string `json:"url"`
		} `json:"primary"`
	} `json:"resource"`
	Subject  []string `json:"subject"`
	Subtitle []string `json:"subtitle"`
	Title    []string `json:"title"`
	Url      string   `json:"url"`
	Version  string   `json:"version"`
	Volume   string   `json:"volume"`
}

// source: http://api.crossref.org/types
var CRToCMMappings = map[string]string{
	"book-chapter":        "BookChapter",
	"book-part":           "BookPart",
	"book-section":        "BookSection",
	"book-series":         "BookSeries",
	"book-set":            "BookSet",
	"book-track":          "BookTrack",
	"book":                "Book",
	"component":           "Component",
	"database":            "Database",
	"dataset":             "Dataset",
	"dissertation":        "Dissertation",
	"edited-book":         "Book",
	"grant":               "Grant",
	"journal-article":     "JournalArticle",
	"journal-issue":       "JournalIssue",
	"journal-volume":      "JournalVolume",
	"journal":             "Journal",
	"monograph":           "Book",
	"other":               "Other",
	"peer-review":         "PeerReview",
	"posted-content":      "Article",
	"proceedings-article": "ProceedingsArticle",
	"proceedings-series":  "ProceedingsSeries",
	"proceedings":         "Proceedings",
	"reference-book":      "Book",
	"reference-entry":     "Entry",
	"report-component":    "ReportComponent",
	"report-series":       "ReportSeries",
	"report":              "Report",
	"standard":            "Standard",
}

var CrossrefContainerTypes = map[string]string{
	"book-chapter":        "book",
	"dataset":             "database",
	"journal-article":     "journal",
	"journal-issue":       "journal",
	"monograph":           "book-series",
	"proceedings-article": "proceedings",
	"posted-content":      "periodical",
}

var CRToCMContainerTranslations = map[string]string{
	"book":        "Book",
	"book-series": "BookSeries",
	"database":    "DataRepository",
	"journal":     "Journal",
	"proceedings": "Proceedings",
	"periodical":  "Periodical",
}

// relation types to include
var relationTypes = []string{"IsPartOf", "HasPart", "IsVariantFormOf", "IsOriginalFormOf", "IsIdenticalTo", "IsTranslationOf", "IsReviewedBy", "Reviews", "HasReview", "IsPreprintOf", "HasPreprint", "IsSupplementTo", "IsSupplementedBy"}

func FetchCrossref(str string) (types.Data, error) {
	var data types.Data
	id, ok := doiutils.ValidateDOI(str)
	if !ok {
		return data, errors.New("invalid DOI")
	}
	content, err := GetCrossref(id)
	if err != nil {
		return data, err
	}
	data, err = ReadCrossref(content)
	if err != nil {
		return data, err
	}
	return data, nil
}

func FetchCrossrefSample(number int, member string, _type string, hasORCID bool, hasROR bool, hasReferences bool, hasRelation bool, hasAbstract bool, hasAward bool, hasLicense bool, hasArchive bool) ([]types.Data, error) {

	var data []types.Data
	content, err := GetCrossrefSample(number, member, _type, hasORCID, hasROR, hasReferences, hasRelation, hasAbstract, hasAward, hasLicense, hasArchive)
	if err != nil {
		return data, err
	}
	for _, v := range content {
		d, err := ReadCrossref(v)
		if err != nil {
			log.Println(err)
		}
		data = append(data, d)
	}
	return data, nil
}

func GetCrossref(pid string) (Content, error) {
	// the envelope for the JSON response from the Crossref API
	type Response struct {
		Status         string  `json:"status"`
		MessageType    string  `json:"message-type"`
		MessageVersion string  `json:"message-version"`
		Message        Content `json:"message"`
	}

	var response Response
	doi, ok := doiutils.ValidateDOI(pid)
	if !ok {
		return response.Message, errors.New("invalid DOI")
	}
	url := "https://api.crossref.org/works/" + doi
	req, err := http.NewRequest("GET", url, nil)
	v := "0.1"
	u := "info@front-matter.io"
	userAgent := fmt.Sprintf("commonmeta-go/%s (https://commonmeta.org/commonmeta-go/; mailto: %s)", v, u)
	req.Header.Set("User-Agent", userAgent)
	if err != nil {
		log.Fatalln(err)
	}
	client := http.Client{
		Timeout: 10 * time.Second,
	}
	resp, err := client.Do(req)
	if err != nil {
		return response.Message, err
	}
	if resp.StatusCode >= 400 {
		return response.Message, errors.New(resp.Status)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return response.Message, err
	}
	err = json.Unmarshal(body, &response)
	if err != nil {
		fmt.Println("error:", err)
	}
	return response.Message, err
}

// read Crossref JSON response and return work struct in Commonmeta format
func ReadCrossref(content Content) (types.Data, error) {
	var data = types.Data{}

	data.ID = doiutils.NormalizeDOI(content.DOI)
	data.Type = CRToCMMappings[content.Type]
	data.Identifiers = append(data.Identifiers, types.Identifier{
		Identifier:     data.ID,
		IdentifierType: "DOI",
	})
	data.Url = content.Resource.Primary.URL

	for _, v := range content.Author {
		if v.Name != "" || v.Given != "" || v.Family != "" {
			var ID, Type string
			if v.ORCID != "" {
				// enforce HTTPS
				ID, _ = utils.NormalizeUrl(v.ORCID, true, false)
			}
			if v.Name != "" {
				Type = "Organization"
			} else {
				Type = "Person"
			}
			var affiliations []types.Affiliation
			if len(v.Affiliation) > 0 {
				for _, a := range v.Affiliation {
					var ID string
					if len(a.ID) > 0 && a.ID[0].IDType == "ROR" {
						ID = utils.NormalizeROR(a.ID[0].ID)
					}
					affiliations = append(affiliations, types.Affiliation{
						ID:   ID,
						Name: a.Name,
					})
				}
			}
			data.Contributors = append(data.Contributors, types.Contributor{
				ID:               ID,
				Type:             Type,
				GivenName:        v.Given,
				FamilyName:       v.Family,
				Name:             v.Name,
				ContributorRoles: []string{"Author"},
				Affiliations:     affiliations,
			})

		}
	}

	if len(content.Title) > 0 && content.Title[0] != "" {
		data.Titles = append(data.Titles, types.Title{
			Title: content.Title[0],
		})
	}
	if len(content.Subtitle) > 0 {
		data.Titles = append(data.Titles, types.Title{
			Title: content.Subtitle[0],
			Type:  "Subtitle",
		})
	}
	if len(content.OriginalTitle) > 0 {
		data.Titles = append(data.Titles, types.Title{
			Title: content.OriginalTitle[0],
			Type:  "TranslatedTitle",
		})
	}

	if content.Publisher != "" {
		data.Publisher = types.Publisher{
			Name: content.Publisher,
		}
	}

	if content.Published.DateTime != "" {
		data.Date.Published = content.Published.DateTime
	} else if len(content.Published.DateAsParts) > 0 {
		data.Date.Published = dateutils.GetDateFromDateParts(content.Published.DateAsParts)
	} else if content.Issued.DateTime != "" {
		data.Date.Published = content.Issued.DateTime
	} else if len(content.Issued.DateAsParts) > 0 {
		data.Date.Published = dateutils.GetDateFromDateParts(content.Issued.DateAsParts)
	}
	if data.Date.Published == "" {
		if content.Created.DateTime != "" {
			data.Date.Published = content.Created.DateTime
		} else if len(content.Created.DateAsParts) > 0 {
			data.Date.Published = dateutils.GetDateFromDateParts(content.Created.DateAsParts)
		}
	}

	containerType := CrossrefContainerTypes[content.Type]
	containerType = CRToCMContainerTranslations[containerType]
	var identifier, identifierType string
	if content.ISSN != nil {
		identifier = content.ISSN[0]
		identifierType = "ISSN"
	}
	if len(content.ISBNType) > 0 {
		i := make(map[string]string)
		for _, isbn := range content.ISBNType {
			i[isbn.Type] = isbn.Value
		}
		if i["electronic"] != "" {
			identifier = i["electronic"]
			identifierType = "ISBN"
		} else if i["print"] != "" {
			identifier = i["print"]
			identifierType = "ISBN"
		}
	}
	var containerTitle string
	if len(content.ContainerTitle) > 0 {
		containerTitle = content.ContainerTitle[0]
	}
	var lastPage string
	pages := strings.Split(content.Page, "-")
	firstPage := pages[0]
	if len(pages) > 1 {
		lastPage = pages[1]
	}
	data.Container = types.Container{
		Identifier:     identifier,
		IdentifierType: identifierType,
		Type:           containerType,
		Title:          containerTitle,
		Volume:         content.Volume,
		Issue:          content.Issue,
		FirstPage:      firstPage,
		LastPage:       lastPage,
	}

	if content.Abstract != "" {
		abstract := utils.Sanitize(content.Abstract)
		data.Descriptions = append(data.Descriptions, types.Description{
			Description: abstract,
			Type:        "Abstract",
		})
	}

	for _, v := range content.Subject {
		subject := types.Subject{
			Subject: v,
		}
		if !slices.Contains(data.Subjects, subject) {
			data.Subjects = append(data.Subjects, subject)
		}
	}

	if content.GroupTitle != "" {
		data.Subjects = append(data.Subjects, types.Subject{
			Subject: content.GroupTitle,
		})
	}

	data.Language = content.Language
	if content.License != nil && len(content.License) > 0 {
		url, _ := utils.NormalizeCCUrl(content.License[0].Url)
		id := utils.UrlToSPDX(url)
		data.License = types.License{
			ID:  id,
			Url: url,
		}
	}

	for _, v := range content.Reference {
		data.References = append(data.References, types.Reference{
			Key:             v.Key,
			ID:              doiutils.NormalizeDOI(v.DOI),
			Title:           v.ArticleTitle,
			PublicationYear: v.Year,
			Unstructured:    v.Unstructured,
		})
	}

	fields := reflect.VisibleFields(reflect.TypeOf(content.Relation))
	for _, field := range fields {
		if slices.Contains(relationTypes, field.Name) {
			relationByType := reflect.ValueOf(content.Relation).FieldByName(field.Name)
			for _, v := range relationByType.Interface().([]struct {
				ID     string `json:"id"`
				IDType string `json:"id-type"`
			}) {
				data.Relations = append(data.Relations, types.Relation{
					ID:   doiutils.NormalizeDOI(v.ID),
					Type: field.Name,
				})
			}
			sort.Slice(data.Relations, func(i, j int) bool {
				return data.Relations[i].Type < data.Relations[j].Type
			})
		}
	}
	if content.ISSN != nil {
		data.Relations = append(data.Relations, types.Relation{
			ID:   utils.IssnAsUrl(content.ISSN[0]),
			Type: "IsPartOf",
		})
	}

	for _, v := range content.Funder {
		funderIdentifier := doiutils.NormalizeDOI(v.DOI)
		var funderIdentifierType string
		if strings.HasPrefix(v.DOI, "10.13039") {
			funderIdentifierType = "Crossref Funder ID"
		}
		if len(v.Award) > 0 {
			for _, award := range v.Award {
				data.FundingReferences = append(data.FundingReferences, types.FundingReference{
					FunderIdentifier:     funderIdentifier,
					FunderIdentifierType: funderIdentifierType,
					FunderName:           v.Name,
					AwardNumber:          award,
				})
			}
		} else {
			data.FundingReferences = append(data.FundingReferences, types.FundingReference{
				FunderIdentifier:     funderIdentifier,
				FunderIdentifierType: funderIdentifierType,
				FunderName:           v.Name,
			})
		}
	}
	data.FundingReferences = utils.DedupeSlice(data.FundingReferences)

	for _, v := range content.Link {
		if v.ContentType != "unspecified" {
			data.Files = append(data.Files, types.File{
				Url:      v.Url,
				MimeType: v.ContentType,
			})
		}
	}
	data.Files = utils.DedupeSlice(data.Files)

	if len(content.Archive) > 0 {
		data.ArchiveLocations = append(data.ArchiveLocations, content.Archive...)
	}

	data.Provider = "Crossref"

	return data, nil
}

func GetCrossrefSample(number int, member string, _type string, hasORCID bool, hasROR bool, hasReferences bool, hasRelation bool, hasAbstract bool, hasAward bool, hasLicense bool, hasArchive bool) ([]Content, error) {
	// the envelope for the JSON response from the Crossref API
	type Response struct {
		Status         string `json:"status"`
		MessageType    string `json:"message-type"`
		MessageVersion string `json:"message-version"`
		Message        struct {
			TotalResults int       `json:"total-results"`
			Items        []Content `json:"items"`
		} `json:"message`
	}
	var response Response
	if number > 100 {
		number = 100
	}
	url := CrossrefApiSampleUrl(number, member, _type, hasORCID, hasROR, hasReferences, hasRelation, hasAbstract, hasAward, hasLicense, hasArchive)
	req, err := http.NewRequest("GET", url, nil)
	v := "0.1"
	u := "info@front-matter.io"
	userAgent := fmt.Sprintf("commonmeta-go/%s (https://commonmeta.org/commonmeta-go/; mailto: %s)", v, u)
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Cache-Control", "private")
	if err != nil {
		log.Fatalln(err)
	}
	client := http.Client{
		Timeout: 20 * time.Second,
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return nil, errors.New(resp.Status)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(body, &response)
	if err != nil {
		fmt.Println("error:", err)
	}
	log.Println("Total results:", response.Message.TotalResults)
	return response.Message.Items, nil
}

func CrossrefApiSampleUrl(number int, member string, _type string, hasORCID bool, hasROR bool, hasReferences bool, hasRelation bool, hasAbstract bool, hasAward bool, hasLicense bool, hasArchive bool) string {
	types := []string{
		"book-section",
		"monograph",
		"report-component",
		"report",
		"peer-review",
		"book-track",
		"journal-article",
		"book-part",
		"other",
		"book",
		"journal-volume",
		"book-set",
		"reference-entry",
		"proceedings-article",
		"journal",
		"component",
		"book-chapter",
		"proceedings-series",
		"report-series",
		"proceedings",
		"database",
		"standard",
		"reference-book",
		"posted-content",
		"journal-issue",
		"dissertation",
		"grant",
		"dataset",
		"book-series",
		"edited-book",
		"journal-section",
		"monograph-series",
		"journal-meta",
		"book-series-meta",
		"component-list",
		"journal-issue-meta",
		"journal-meta",
		"book-part-meta",
		"book-meta",
		"proceedings-meta",
		"book-series-meta",
		"book-set",
	}
	u, _ := url.Parse("https://api.crossref.org/works")
	values := u.Query()
	values.Add("sample", strconv.Itoa(number))
	var filters []string
	if member != "" {
		filters = append(filters, "member:"+member)
	}
	if _type != "" && slices.Contains(types, _type) {
		filters = append(filters, "type:"+_type)
	}
	if hasORCID {
		filters = append(filters, "has-orcid:true")
	}
	if hasROR {
		filters = append(filters, "has-ror-id:true")
	}
	if hasReferences {
		filters = append(filters, "has-references:true")
	}
	if hasRelation {
		filters = append(filters, "has-relation:true")
	}
	if hasAbstract {
		filters = append(filters, "has-abstract:true")
	}
	if hasAward {
		filters = append(filters, "has-award:true")
	}
	if hasLicense {
		filters = append(filters, "has-license:true")
	}
	if hasArchive {
		filters = append(filters, "has-archive:true")
	}
	if len(filters) > 0 {
		values.Add("filter", strings.Join(filters[:], ","))
	}
	u.RawQuery = values.Encode()
	return u.String()
}
