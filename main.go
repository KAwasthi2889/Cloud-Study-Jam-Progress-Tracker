package main

import (
	"encoding/csv"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

var requiredBadges = []string{
	"The Basics of Google Cloud Compute",
	"Get Started with Cloud Storage",
	"Get Started with Pub/Sub",
	"Get Started with API Gateway",
	"Get Started with Looker",
	"Get Started with Dataplex",
	"Get Started with Google Workspace Tools",
	"App Building with Appsheet",
	"Develop with Apps Script and AppSheet",
	"Build a Website on Google Cloud",
	"Set Up a Google Cloud Network",
	"Store, Process, and Manage Data on Google Cloud - Console",
	"Cloud Run Functions: 3 Ways",
	"App Engine: 3 Ways",
	"Cloud Speech API: 3 Ways",
	"Monitoring in Google Cloud",
	"Analyze Speech and Language with Google APIs",
	"Prompt Design in Vertex AI",
	"Develop Gen AI Apps with Gemini and Streamlit",
	"Level 3: Generative AI",
}

// normalize returns a simplified lowercase version of the string.
func normalize(s string) string {
	s = strings.ToLower(s)
	re := regexp.MustCompile(`[^a-z0-9]+`)
	return re.ReplaceAllString(s, "")
}

// capitalizeWords makes the first letter of each word uppercase.
func capitalizeWords(s string) string {
	words := strings.Fields(s)
	for i, w := range words {
		words[i] = strings.Title(strings.ToLower(w))
	}
	return strings.Join(words, " ")
}

// getBadges scrapes badges and participant name from a profile URL.
func getBadges(profileURL string) ([]string, string, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(profileURL)
	if err != nil {
		return nil, "", fmt.Errorf("failed to fetch profile: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("failed to parse HTML: %v", err)
	}

	// Extract participant name
	name := strings.TrimSpace(doc.Find("h1.ql-display-small").First().Text())
	if name != "" {
		name = capitalizeWords(name)
	}

	var badges []string
	doc.Find("div.profile-badge span.ql-title-medium").Each(func(i int, s *goquery.Selection) {
		title := strings.TrimSpace(s.Text())
		if title != "" {
			badges = append(badges, title)
		}
	})

	return badges, name, nil
}

func main() {
	start := time.Now()

	file, err := os.Open("participants.csv")
	if err != nil {
		log.Fatalf("Failed to open CSV: %v", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.TrimLeadingSpace = true
	records, err := reader.ReadAll()
	if err != nil {
		log.Fatalf("Failed to read CSV: %v", err)
	}

	requiredMap := make(map[string]bool)
	for _, name := range requiredBadges {
		requiredMap[normalize(name)] = true
	}

	var completedAll, someLabsLeft, onlyArcadeLeft [][]string

	for i, row := range records {
		if i == 0 {
			continue
		}
		if len(row) < 7 {
			continue
		}

		csvName := strings.TrimSpace(row[0])
		url := strings.TrimSpace(row[6])
		if url == "" {
			continue
		}

		fmt.Printf("Checking profile: %s\n", csvName)

		badges, scrapedName, err := getBadges(url)
		if err != nil {
			fmt.Printf("  Error fetching profile: %v\n", err)
			continue
		}

		name := csvName
		if scrapedName != "" {
			name = scrapedName
		} else {
			name = capitalizeWords(csvName)
		}

		userBadges := make(map[string]bool)
		for _, b := range badges {
			userBadges[normalize(b)] = true
		}

		total := 0
		hasLevel3 := false
		for k := range userBadges {
			if requiredMap[k] {
				total++
				if k == normalize("Level 3: Generative AI") {
					hasLevel3 = true
				}
			}
		}

		switch {
		case total == 20:
			completedAll = append(completedAll, []string{name, url})
		case hasLevel3 && total > 13:
			someLabsLeft = append(someLabsLeft, []string{name, url})
		case !hasLevel3 && total == 19:
			onlyArcadeLeft = append(onlyArcadeLeft, []string{name, url})
		}
	}

	// Sort alphabetically by name
	sort.Slice(completedAll, func(i, j int) bool { return completedAll[i][0] < completedAll[j][0] })
	sort.Slice(someLabsLeft, func(i, j int) bool { return someLabsLeft[i][0] < someLabsLeft[j][0] })
	sort.Slice(onlyArcadeLeft, func(i, j int) bool { return onlyArcadeLeft[i][0] < onlyArcadeLeft[j][0] })

	fmt.Printf("\n===== RESULTS =====\n")
	fmt.Printf("Completed Everything: %d\n", len(completedAll))
	fmt.Printf("Some Labs Left: %d\n", len(someLabsLeft))
	fmt.Printf("Only Arcade Left: %d\n", len(onlyArcadeLeft))

	out, err := os.Create("results.csv")
	if err != nil {
		log.Fatalf("Failed to create results.csv: %v", err)
	}
	defer out.Close()
	writer := csv.NewWriter(out)

	writeGroup := func(title string, entries [][]string) {
		if len(entries) == 0 {
			return
		}
		writer.Write([]string{fmt.Sprintf("# %s (%d)", title, len(entries))})
		writer.Write([]string{"Name", "Profile URL"})
		for _, row := range entries {
			writer.Write(row)
		}
		writer.Write([]string{})
	}

	writeGroup("Completed Everything", completedAll)
	writeGroup("Some Labs Left", someLabsLeft)
	writeGroup("Only Arcade Left", onlyArcadeLeft)
	writer.Flush()

	fmt.Printf("\nProgram finished in %v\n", time.Since(start).Round(time.Second))
}
