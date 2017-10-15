package main

import (
	"flag"
	"fmt"
	"log"
	"math"
	"os"

	"github.com/dhowden/tag"
)

var (
	filename string
	verbose  bool
)

const appleOwner string = "com.apple.streaming.transportStreamTimestamp"

func init() {
	flag.StringVar(&filename, "filename", "", "File to parse id3 tags from")
	flag.BoolVar(&verbose, "v", false, "Enable verbose logging output")
}

func main() {
	flag.Parse()
	if verbose {
		log.Println("Trying to open " + filename)
	}
	f, err := os.Open(filename)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	m, err := tag.ReadFrom(f)
	if err != nil {
		log.Fatal(err)
	}
	if verbose {
		log.Printf("Detected format: %v", m.Format()) // The detected format.
	}
	rawTags := m.Raw()
	if len(rawTags) != 0 {
		if val, ok := rawTags["PRIV"]; ok {
			tagOwner := string([]byte(val.([]uint8))[0:44])
			if tagOwner == appleOwner {
				timestampData := []byte(val.([]uint8))[45:]
				if verbose {
					log.Println("Found apple PRIV tag")
					log.Printf("Bytes found: %v\n", timestampData)
				}

				// Adapted from hls.js
				// How it works is there is 64 bits represented by timestampData
				// the hls spec mandates that the upper 31 bits are set to 0
				// so for example if you have {0x0, 0x0, 0x0, 0x0, 0x82, 0x13, 0x9e, 0xf8}
				// 00000000 00000000 00000000 00000000
				// 10000010 00010011 10011110 11111000
				// the first 3 0x0's and the upper 7 bits of the last 0x0 are set to 0 for the spec
				// the remaining bits represent a 31 bit integer:
				// 01000001 00001001 11001111 01111100 0 (spaces for clarity)
				// thus we bit shift everything (multiply by the power of 2 that it represents)
				pts33Bit := timestampData[3] & 0x1
				timestamp := float64(float64(timestampData[4])*math.Pow(2, 23)) +
					float64(float64(timestampData[5])*math.Pow(2, 15)) +
					float64(float64(timestampData[6])*math.Pow(2, 7)) +
					float64(timestampData[7])
				timestamp *= 2
				if verbose {
					log.Printf("Uppermost bit: %v\n", pts33Bit)
				}

				if pts33Bit == 0x1 {
					timestamp += math.Pow(2, 31)
					// timestamp += 47721858.84 // 2^32 / 90
				}
				fmt.Printf("{ \"start_pts\":%d }", int64(timestamp))
				return
			}
		}
	}

}
