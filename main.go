package main

import (
	"encoding/csv"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
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
	"Cloud Run Functions: 3 Ways", // ✅ Updated name
	"App Engine: 3 Ways",
	"Cloud Speech API: 3 Ways",
	"Monitoring in Google Cloud",
	"Analyze Speech and Language with Google APIs",
	"Prompt Design in Vertex AI",
	"Develop Gen AI Apps with Gemini and Streamlit", // ✅ Updated name
	"Level 3: Generative AI",
}

func normalize(s string) string {
	s = strings.ToLower(s)
	re := regexp.MustCompile(`[^a-z0-9]+`)
	return re.ReplaceAllString(s, "")
}

// Scrape badge names from a user's Skills Boost profile
func getBadges(profileURL string) ([]string, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(profileURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch profile: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %v", err)
	}

	var badges []string
	doc.Find("div.profile-badge span.ql-title-medium").Each(func(i int, s *goquery.Selection) {
		title := strings.TrimSpace(s.Text())
		if title != "" {
			badges = append(badges, title)
		}
	})
	return badges, nil
}

func main() {
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

	completedAll := []string{}
	someLabsLeft := []string{}
	onlyArcadeLeft := []string{}

	for i, row := range records {
		if i == 0 {
			continue // skip header
		}
		if len(row) < 7 {
			continue
		}

		name := strings.TrimSpace(row[0])
		url := strings.TrimSpace(row[6])
		if url == "" {
			continue
		}

		fmt.Printf("\n🔹 Checking %s\n", name)

		badges, err := getBadges(url)
		if err != nil {
			fmt.Printf("  ❌ Error fetching profile: %v\n", err)
			continue
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

		fmt.Printf("  ✅ Total relevant badges: %d\n", total)

		switch {
		case total == 20:
			completedAll = append(completedAll, name)
		case hasLevel3 && total > 13:
			someLabsLeft = append(someLabsLeft, name)
		case !hasLevel3 && total == 19:
			onlyArcadeLeft = append(onlyArcadeLeft, name)
		}

		time.Sleep(2 * time.Second) // gentle delay
	}

	fmt.Printf("\n\n===== RESULTS =====\n")
	fmt.Printf("\nCompleted everything (%d):\n", len(completedAll))
	for _, n := range completedAll {
		fmt.Printf("  - %s\n", n)
	}

	fmt.Printf("\nSome Labs Left (%d):\n", len(someLabsLeft))
	for _, n := range someLabsLeft {
		fmt.Printf("  - %s\n", n)
	}

	fmt.Printf("\nOnly arcade left (%d):\n", len(onlyArcadeLeft))
	for _, n := range onlyArcadeLeft {
		fmt.Printf("  - %s\n", n)
	}
}
