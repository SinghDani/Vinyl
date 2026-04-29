package fingerprint

/*
func exportAllHistogramsToCSV(timedMatches map[deltaKey]int, filename string) {
	file, err := os.Create(filename)
	if err != nil {
		fmt.Println("Error creating CSV:", err)
		return
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	writer.Write([]string{"song_id", "dt", "count"})

	for key, count := range timedMatches {
		if count > 1 {
			row := []string{
				strconv.FormatUint(uint64(key.song_id), 10),
				strconv.FormatInt(key.dt, 10),
				strconv.Itoa(count),
			}
			writer.Write(row)
		}
	}
	fmt.Println("Saved full histogram to", filename)
}

*/
