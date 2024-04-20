package datacite

import (
	"commonmeta/doiutils"
	"commonmeta/types"
	"commonmeta/utils"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"
)

type Content struct {
	ID         string     `json:"id"`
	Type       string     `json:"type"`
	Attributes Attributes `json:"attributes"`
}

type Attributes struct {
	DOI                  string `json:"doi"`
	Prefix               string `json:"prefix"`
	Suffix               string `json:"suffix"`
	AlternateIdentifiers []struct {
		Identifier     string `json:"identifier"`
		IdentifierType string `json:"identifierType"`
	} `json:"alternateIdentifiers"`
	Creators  []Contributor `json:"creators"`
	Publisher string        `json:"publisher"`
	Container struct {
		Type           string `json:"type"`
		Identifier     string `json:"identifier"`
		IdentifierType string `json:"identifierType"`
		Title          string `json:"title"`
		Volume         string `json:"volume"`
		Issue          string `json:"issue"`
		FirstPage      string `json:"firstPage"`
		LastPage       string `json:"lastPage"`
	} `json:"container"`
	PublicationYear int `json:"publicationYear"`
	Titles          []struct {
		Title     string `json:"title"`
		TitleType string `json:"titleType"`
		Lang      string `json:"lang"`
	} `json:"titles"`
	Url      string `json:"url"`
	Subjects []struct {
		Subject string `json:"subject"`
	} `json:"subjects"`
	Contributors []Contributor `json:"contributors"`
	Dates        []struct {
		Date            string `json:"date"`
		DateType        string `json:"dateType"`
		DateInformation string `json:"dateInformation"`
	} `json:"dates"`
	Language string `json:"language"`
	Types    struct {
		ResourceTypeGeneral string `json:"resourceTypeGeneral"`
		ResourceType        string `json:"resourceType"`
	} `json:"types"`
	RelatedIdentifiers []struct {
		RelatedIdentifier     string `json:"relatedIdentifier"`
		RelatedIdentifierType string `json:"relatedIdentifierType"`
		RelationType          string `json:"relationType"`
	} `json:"relatedIdentifiers"`
	Sizes      []string `json:"sizes"`
	Formats    []string `json:"formats"`
	Version    string   `json:"version"`
	RightsList []struct {
		Rights                 string `json:"rights"`
		RightsURI              string `json:"rightsUri"`
		SchemeURI              string `json:"schemeUri"`
		RightsIdentifier       string `json:"rightsIdentifier"`
		RightsIdentifierScheme string `json:"rightsIdentifierScheme"`
	}
	Descriptions []struct {
		Description     string `json:"description"`
		DescriptionType string `json:"descriptionType"`
		Lang            string `json:"lang"`
	} `json:"descriptions"`
	GeoLocations []struct {
		GeoLocationPoint struct {
			PointLongitude float64 `json:"pointLongitude,string"`
			PointLatitude  float64 `json:"pointLatitude,string"`
		} `json:"geoLocationPoint"`
		GeoLocationBox struct {
			WestBoundLongitude float64 `json:"westBoundLongitude,string"`
			EastBoundLongitude float64 `json:"eastBoundLongitude,string"`
			SouthBoundLatitude float64 `json:"southBoundLatitude,string"`
			NorthBoundLatitude float64 `json:"northBoundLatitude,string"`
		} `json:"geoLocationBox"`
		GeoLocationPlace string `json:"geoLocationPlace"`
	} `json:"geoLocations"`
	FundingReferences []struct {
		FunderName           string `json:"funderName"`
		FunderIdentifier     string `json:"funderIdentifier"`
		FunderIdentifierType string `json:"funderIdentifierType"`
		AwardNumber          string `json:"awardNumber"`
		AwardURI             string `json:"awardUri"`
	} `json:"fundingReferences"`
}

type Contributor struct {
	Name            string `json:"name"`
	GivenName       string `json:"givenName"`
	FamilyName      string `json:"familyName"`
	NameType        string `json:"nameType"`
	NameIdentifiers []struct {
		SchemeURI            string `json:"schemeUri"`
		NameIdentifier       string `json:"nameIdentifier"`
		NameIdentifierScheme string `json:"nameIdentifierScheme"`
	} `json:"nameIdentifiers"`
	Affiliation     []string `json:"affiliation"`
	ContributorType string   `json:"contributorType"`
}

// source: https://github.com/datacite/schema/blob/master/source/meta/kernel-4/include/datacite-resourceType-v4.xsd
var DCToCMTranslations = map[string]string{
	"Audiovisual":           "Audiovisual",
	"BlogPosting":           "Article",
	"Book":                  "Book",
	"BookChapter":           "BookChapter",
	"Collection":            "Collection",
	"ComputationalNotebook": "ComputationalNotebook",
	"ConferencePaper":       "ProceedingsArticle",
	"ConferenceProceeding":  "Proceedings",
	"DataPaper":             "JournalArticle",
	"Dataset":               "Dataset",
	"Dissertation":          "Dissertation",
	"Event":                 "Event",
	"Image":                 "Image",
	"Instrument":            "Instrument",
	"InteractiveResource":   "InteractiveResource",
	"Journal":               "Journal",
	"JournalArticle":        "JournalArticle",
	"Model":                 "Model",
	"OutputManagementPlan":  "OutputManagementPlan",
	"PeerReview":            "PeerReview",
	"PhysicalObject":        "PhysicalObject",
	"Poster":                "Presentation",
	"Preprint":              "Article",
	"Report":                "Report",
	"Service":               "Service",
	"Software":              "Software",
	"Sound":                 "Sound",
	"Standard":              "Standard",
	"StudyRegistration":     "StudyRegistration",
	"Text":                  "Document",
	"Thesis":                "Dissertation",
	"Workflow":              "Workflow",
	"Other":                 "Other",
}

// from commonmeta schema
var CommonmetaContributorRoles = []string{
	"Author",
	"Editor",
	"Chair",
	"Reviewer",
	"ReviewAssistant",
	"StatsReviewer",
	"ReviewerExternal",
	"Reader",
	"Translator",
	"ContactPerson",
	"DataCollector",
	"DataManager",
	"Distributor",
	"HostingInstitution",
	"Producer",
	"ProjectLeader",
	"ProjectManager",
	"ProjectMember",
	"RegistrationAgency",
	"RegistrationAuthority",
	"RelatedPerson",
	"ResearchGroup",
	"RightsHolder",
	"Researcher",
	"Sponsor",
	"WorkPackageLeader",
	"Conceptualization",
	"DataCuration",
	"FormalAnalysis",
	"FundingAcquisition",
	"Investigation",
	"Methodology",
	"ProjectAdministration",
	"Resources",
	"Software",
	"Supervision",
	"Validation",
	"Visualization",
	"WritingOriginalDraft",
	"WritingReviewEditing",
	"Maintainer",
	"Other",
}

func FetchDatacite(str string) (types.Data, error) {
	var data types.Data
	id, ok := doiutils.ValidateDOI(str)
	if !ok {
		return data, errors.New("invalid doi")
	}
	content, err := GetDatacite(id)
	if err != nil {
		return data, err
	}
	data, err = ReadDatacite(content)
	if err != nil {
		return data, err
	}
	return data, nil
}

func FetchDataciteSample(number int) ([]types.Data, error) {

	var data []types.Data
	content, err := GetDataciteSample(number)
	if err != nil {
		return data, err
	}
	for _, v := range content {
		d, err := ReadDatacite(v)
		if err != nil {
			log.Println(err)
		}
		data = append(data, d)
	}
	return data, nil
}

func GetDatacite(pid string) (Content, error) {
	// the envelope for the JSON response from the DataCite API
	type Response struct {
		Data Content `json:"data"`
	}

	var response Response
	doi, ok := doiutils.ValidateDOI(pid)
	if !ok {
		return response.Data, errors.New("Invalid DOI")
	}
	url := "https://api.datacite.org/dois/" + doi
	client := http.Client{
		Timeout: time.Second * 10,
	}
	resp, err := client.Get(url)
	if err != nil {
		return response.Data, err
	}
	if resp.StatusCode >= 400 {
		return response.Data, errors.New(resp.Status)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return response.Data, err
	}
	err = json.Unmarshal(body, &response)
	if err != nil {
		fmt.Println("error:", err)
	}
	return response.Data, err
}

// read DataCite JSON response and return work struct in Commonmeta format
func ReadDatacite(content Content) (types.Data, error) {
	var data = types.Data{}

	data.ID = doiutils.NormalizeDOI(content.Attributes.DOI)
	data.Type = DCToCMTranslations[content.Attributes.Types.ResourceTypeGeneral]
	var err error

	data.Identifiers = append(data.Identifiers, types.Identifier{
		Identifier:     data.ID,
		IdentifierType: "DOI",
	})
	if len(content.Attributes.AlternateIdentifiers) > 0 {
		for _, v := range content.Attributes.AlternateIdentifiers {
			if content.Attributes.AlternateIdentifiers[0].Identifier != "" {
				data.Identifiers = append(data.Identifiers, types.Identifier{
					Identifier:     v.Identifier,
					IdentifierType: v.IdentifierType,
				})
			}
		}
	}

	data.AdditionalType = DCToCMTranslations[content.Attributes.Types.ResourceType]
	if data.AdditionalType != "" {
		data.Type = data.AdditionalType
		data.AdditionalType = ""
	} else {
		data.AdditionalType = content.Attributes.Types.ResourceType
	}
	data.Url, err = utils.NormalizeUrl(content.Attributes.Url, true, false)
	if err != nil {
		log.Println(err)
	}

	if len(content.Attributes.Creators) > 0 {
		for _, v := range content.Attributes.Creators {
			if v.Name != "" || v.GivenName != "" || v.FamilyName != "" {
				contributor := GetContributor(v)
				containsID := slices.ContainsFunc(data.Contributors, func(e types.Contributor) bool {
					return e.ID != "" && e.ID == contributor.ID
				})
				if containsID {
					log.Printf("Contributor with ID %s already exists", contributor.ID)
				} else {
					data.Contributors = append(data.Contributors, contributor)

				}
			}
		}

		// merge creators and contributors
		for _, v := range content.Attributes.Contributors {
			if v.Name != "" || v.GivenName != "" || v.FamilyName != "" {
				contributor := GetContributor(v)
				containsID := slices.ContainsFunc(data.Contributors, func(e types.Contributor) bool {
					return e.ID != "" && e.ID == contributor.ID
				})
				if containsID {
					log.Printf("Contributor with ID %s already exists", contributor.ID)
				} else {
					data.Contributors = append(data.Contributors, contributor)

				}
			}
		}
	}

	if len(content.Attributes.Titles) > 0 {
		for _, v := range content.Attributes.Titles {
			var t string
			if slices.Contains([]string{"MainTitle", "Subtitle", "TranslatedTitle"}, v.TitleType) {
				t = v.TitleType
			}
			data.Titles = append(data.Titles, types.Title{
				Title:    v.Title,
				Type:     t,
				Language: v.Lang,
			})
		}
	}

	if content.Attributes.Publisher != "" {
		data.Publisher = types.Publisher{
			Name: content.Attributes.Publisher,
		}
	}

	for _, v := range content.Attributes.Dates {
		if v.DateType == "Accepted" {
			data.Date.Accepted = v.Date
		}
		if v.DateType == "Available" {
			data.Date.Available = v.Date
		}
		if v.DateType == "Collected" {
			data.Date.Collected = v.Date
		}
		if v.DateType == "Copyrighted" {
			data.Date.Copyrighted = v.Date
		}
		if v.DateType == "Created" {
			data.Date.Created = v.Date
		}
		if v.DateType == "Issued" {
			data.Date.Published = v.Date
		} else if v.DateType == "Published" {
			data.Date.Published = v.Date
		}
		if v.DateType == "Submitted" {
			data.Date.Submitted = v.Date
		}
		if v.DateType == "Updated" {
			data.Date.Updated = v.Date
		}
		if v.DateType == "Valid" {
			data.Date.Valid = v.Date
		}
		if v.DateType == "Withdrawn" {
			data.Date.Withdrawn = v.Date
		}
		if v.DateType == "Other" {
			data.Date.Other = v.Date
		}
	}

	data.Container = types.Container{
		Identifier:     content.Attributes.Container.Identifier,
		IdentifierType: content.Attributes.Container.IdentifierType,
		Type:           content.Attributes.Container.Type,
		Title:          content.Attributes.Container.Title,
		Volume:         content.Attributes.Container.Volume,
		Issue:          content.Attributes.Container.Issue,
		FirstPage:      content.Attributes.Container.FirstPage,
		LastPage:       content.Attributes.Container.LastPage,
	}

	if len(content.Attributes.Descriptions) > 0 {
		for _, v := range content.Attributes.Descriptions {
			var t string
			if slices.Contains([]string{"Abstract", "Summary", "Methods", "TechnicalInfo", "Other"}, v.DescriptionType) {
				t = v.DescriptionType
			} else {
				t = "Other"
			}
			description := utils.Sanitize(v.Description)
			data.Descriptions = append(data.Descriptions, types.Description{
				Description: description,
				Type:        t,
				Language:    v.Lang,
			})
		}
	}

	if len(content.Attributes.Subjects) > 0 {
		for _, v := range content.Attributes.Subjects {
			subject := types.Subject{
				Subject: v.Subject,
			}
			if !slices.Contains(data.Subjects, subject) {
				data.Subjects = append(data.Subjects, subject)
			}
		}
	}

	data.Language = content.Attributes.Language

	if len(content.Attributes.RightsList) > 0 {
		url := content.Attributes.RightsList[0].RightsURI
		id := utils.UrlToSPDX(url)
		if id == "" {
			log.Printf("License URL %s not found in SPDX", url)
		}
		data.License = types.License{
			ID:  id,
			Url: url,
		}
	}

	data.Version = content.Attributes.Version

	if len(content.Attributes.RelatedIdentifiers) > 0 {
		supportedRelations := []string{
			"Cites",
			"References",
		}
		for i, v := range content.Attributes.RelatedIdentifiers {
			if slices.Contains(supportedRelations, v.RelationType) {
				id := doiutils.NormalizeDOI(v.RelatedIdentifier)
				if id == "" {
					id = v.RelatedIdentifier
				}
				data.References = append(data.References, types.Reference{
					Key: "ref" + strconv.Itoa(i+1),
					ID:  id,
				})
			}
		}
	}

	if len(content.Attributes.RelatedIdentifiers) > 0 {
		supportedRelations := []string{
			"IsNewVersionOf",
			"IsPreviousVersionOf",
			"IsVersionOf",
			"HasVersion",
			"IsPartOf",
			"HasPart",
			"IsVariantFormOf",
			"IsOriginalFormOf",
			"IsIdenticalTo",
			"IsTranslationOf",
			"IsReviewedBy",
			"Reviews",
			"IsPreprintOf",
			"HasPreprint",
			"IsSupplementTo",
		}
		for _, v := range content.Attributes.RelatedIdentifiers {
			if slices.Contains(supportedRelations, v.RelationType) {
				identifier := doiutils.NormalizeDOI(v.RelatedIdentifier)
				if identifier == "" {
					identifier = v.RelatedIdentifier
				}
				data.Relations = append(data.Relations, types.Relation{
					ID:   identifier,
					Type: v.RelationType,
				})
			}
		}
	}

	if len(content.Attributes.FundingReferences) > 0 {
		for _, v := range content.Attributes.FundingReferences {
			data.FundingReferences = append(data.FundingReferences, types.FundingReference{
				FunderIdentifier:     v.FunderIdentifier,
				FunderIdentifierType: v.FunderIdentifierType,
				FunderName:           v.FunderName,
				AwardNumber:          v.AwardNumber,
				AwardURI:             v.AwardURI,
			})
		}
	} else {
		data.FundingReferences = []types.FundingReference{}
	}

	if len(content.Attributes.GeoLocations) > 0 {
		for _, v := range content.Attributes.GeoLocations {
			data.GeoLocations = append(data.GeoLocations, types.GeoLocation{
				GeoLocationPoint: types.GeoLocationPoint{
					PointLongitude: v.GeoLocationPoint.PointLongitude,
					PointLatitude:  v.GeoLocationPoint.PointLatitude,
				},
				GeoLocationPlace: v.GeoLocationPlace,
				GeoLocationBox: types.GeoLocationBox{
					EastBoundLongitude: v.GeoLocationBox.EastBoundLongitude,
					WestBoundLongitude: v.GeoLocationBox.WestBoundLongitude,
					SouthBoundLatitude: v.GeoLocationBox.SouthBoundLatitude,
					NorthBoundLatitude: v.GeoLocationBox.NorthBoundLatitude,
				},
			})
		}
	}

	data.Files = []types.File{}
	// sizes and formats are part of the file object, but can't be mapped directly

	data.ArchiveLocations = []string{}

	data.Provider = "DataCite"

	return data, nil
}

func GetContributor(v Contributor) types.Contributor {
	var t string
	if len(v.NameType) > 2 {
		t = v.NameType[:len(v.NameType)-2]
	}
	var id string
	if len(v.NameIdentifiers) > 0 {
		ni := v.NameIdentifiers[0]
		if ni.NameIdentifierScheme == "ORCID" || ni.NameIdentifierScheme == "https://orcid.org/" {
			id = utils.NormalizeORCID(ni.NameIdentifier)
			t = "Person"
		} else if ni.NameIdentifierScheme == "ROR" {
			id = ni.NameIdentifier
			t = "Organization"
		} else {
			id = ni.NameIdentifier
		}
	}
	name := v.Name
	GivenName := v.GivenName
	FamilyName := v.FamilyName
	if t == "" && (v.GivenName != "" || v.FamilyName != "") {
		t = "Person"
	} else if t == "" {
		t = "Organization"
	}
	if t == "Person" && name != "" {
		// split name for type Person into given/family name if not already provided
		names := strings.Split(name, ",")
		if len(names) == 2 {
			GivenName = strings.TrimSpace(names[1])
			FamilyName = names[0]
			name = ""
		}
	}
	var affiliations []types.Affiliation
	for _, a := range v.Affiliation {
		affiliations = append(affiliations, types.Affiliation{
			ID:   "",
			Name: a,
		})
	}
	var roles []string
	if slices.Contains(CommonmetaContributorRoles, v.ContributorType) {
		roles = append(roles, v.ContributorType)
	} else {
		roles = append(roles, "Author")
	}
	return types.Contributor{
		ID:               id,
		Type:             t,
		Name:             name,
		GivenName:        GivenName,
		FamilyName:       FamilyName,
		ContributorRoles: roles,
		Affiliations:     affiliations,
	}
}

func GetDataciteSample(number int) ([]Content, error) {
	// the envelope for the JSON response from the DataCite API
	type Response struct {
		Data []Content `json:"data"`
	}
	if number > 100 {
		number = 100
	}
	var response Response
	url := DataciteApiSampleUrl(number)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Fatalln(err)
	}
	client := http.Client{
		Timeout: 60 * time.Second,
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
	return response.Data, nil
}

func DataciteApiSampleUrl(number int) string {
	url := "https://api.datacite.org/dois?random=true&page[size]=" + strconv.Itoa(number)
	return url
}
