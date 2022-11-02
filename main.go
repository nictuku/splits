// Copyright 2022 The golang.design Initiative Authors.
// All rights reserved. Use of this source code is governed
// by a MIT license that can be found in the LICENSE file.

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"golang.design/x/hotkey"
	"io/ioutil"
	"log"
	"os"
	"time"

	_ "go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type SplitsFile struct {
	Splits []string
	Levels []string
}

func readSplits() (*SplitsFile, error) {
	content, err := ioutil.ReadFile("./splits.json")
	if err != nil {
		return nil, err
	}

	var splitsData SplitsFile
	err = json.Unmarshal(content, &splitsData)
	if err != nil {
		return nil, err
	}
	return &splitsData, nil
}

func writeSplits(splitsText []*widget.Label, splitsLevels []string) {
	splitsTimes := []string{}
	for i, s := range splitsText {
		if i%2 == 1 {
			splitsTimes = append(splitsTimes, s.Text)
		}
	}
	data := SplitsFile{
		Splits: splitsTimes,
		Levels: splitsLevels,
	}
	file, _ := json.MarshalIndent(data, "", " ")
	_ = ioutil.WriteFile("splits.json", file, 0644)

}

func formatDuration(d time.Duration) string {
	d = d.Round(time.Second)
	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	d -= m * time.Minute
	s := d / time.Second
	ret := fmt.Sprintf("%02d:%02d:%02d", h, m, s)
	fmt.Println("dur", ret, h, m, s, d)
	return ret
}

func main() {
	ctx := context.Background()
	serverAPIOptions := options.ServerAPI(options.ServerAPIVersion1)
	clientOptions := options.Client().
		ApplyURI(fmt.Sprintf("mongodb+srv://splits:%s@cluster0.x2a6d.mongodb.net/?retryWrites=true&w=majority", os.Getenv("SPLITS_MONGO_PASS"))).
		SetServerAPIOptions(serverAPIOptions)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		log.Fatal(err)
	}
	// Check the connection
	err = client.Ping(context.TODO(), nil)

	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Connected to MongoDB!")

	w := app.New().NewWindow("Mega Man 2 Any% (Normal, Zipless)")
	label := widget.NewLabel("Hello golang.design!")
	button := widget.NewButton("Hi!", func() { label.SetText("Welcome :)") })

	splitsText := []*widget.Label{}
	splits, err := readSplits()
	if err != nil {
		log.Println("Couldn't read splits file, using default ones:", err)
		splits = &SplitsFile{Levels: []string{
			"Flash",
			"Air",
			"Quick",
			"Metal",
			"Bubble",
			"Heat",
			"Wood",
			"Crash",
			"Wily 1",
			"Wily 2",
			"Wily 3",
			"Wily 4",
			"Wily 5",
			"Wily 6",
		}}
		for _, s := range splits.Levels {

			splitTime := widget.NewLabel("00:00")
			splitTime.Alignment = fyne.TextAlignTrailing
			splitsText = append(splitsText, widget.NewLabel(s), splitTime)
		}

	} else if len(splits.Levels) > 0 {
		for i, s := range splits.Splits {

			splitTime := widget.NewLabel(s)
			splitTime.Alignment = fyne.TextAlignTrailing
			// TODO: don't trust that Levels and Splits match
			splitsText = append(splitsText, widget.NewLabel(splits.Levels[i]), splitTime)
		}
	}

	objs := make([]fyne.CanvasObject, 0, len(splitsText)+1)
	for i := range splitsText {
		objs = append(objs, splitsText[i])
	}
	runTimer := widget.NewLabel("0.0")
	runTimer.Alignment = fyne.TextAlignTrailing
	runTimer.TextStyle.Bold = true
	// Keep the first colum of the last row empty
	arrow := widget.NewLabel("==========>  (current PB: 35:22)")
	arrow.Alignment = fyne.TextAlignTrailing
	objs = append(objs, arrow)
	objs = append(objs, runTimer)

	grid := container.New(layout.NewGridLayout(2), objs...)

	w.SetContent(grid)

	go func() {
		// Register a desired hotkey.
		//startKey := hotkey.New([]hotkey.Modifier{hotkey.ModCtrl, hotkey.ModShift}, hotkey.KeyS)

		// I found through trial and error that keypad 8 is 0x005b
		// List of others: https://gist.github.com/eegrok/949034
		// 0x53 is keypad 1, 0x55 is keypad 3
		//
		// My step pedal has keys 3, 8 and 1 (in this order).
		startKey := hotkey.New([]hotkey.Modifier{}, 0x0053)
		if err := startKey.Register(); err != nil {
			panic("hotkey registration failed")
		}
		resetKey := hotkey.New([]hotkey.Modifier{}, 0x0055)
		if err := resetKey.Register(); err != nil {
			panic("hotkey registration failed")
		}

		currentSplit := 0
		var startTime time.Time
		done := make(chan struct{})
		needsReset := false

		// Start listen hotkey event whenever it is ready.
		for {
			select {
			case <-startKey.Keydown():
				if needsReset {
					continue
				}
				fmt.Println("key")
				if startTime.IsZero() {
					startTime = time.Now()
					c := time.NewTicker(time.Second)

					go func() {
						for {
							select {
							case <-c.C:
								tm := time.Since(startTime)
								runTimer.SetText(formatDuration(tm))
								// I think this is unsafe.
								splitsText[currentSplit*2+1].SetText(formatDuration(time.Since(startTime)))
							case <-done:
								c.Stop()
								return
							}
						}
					}()
					continue
				}
				splitsText[currentSplit*2+1].SetText(formatDuration(time.Since(startTime)))
				currentSplit += 1
				button.Tapped(&fyne.PointEvent{})
				if currentSplit >= len(splitsText)/2 {
					runTimer.SetText(fmt.Sprintf("END %v", formatDuration(time.Since(startTime))))
					currentSplit = 0
					startTime = time.Time{}
					select {
					case done <- struct{}{}:
					default:
					}
					needsReset = true
				}
			case <-resetKey.Keydown():
				fmt.Println("reseting")
				writeSplits(splitsText, splits.Levels)
				startTime = time.Time{}
				currentSplit = 0
				fmt.Println("resetting these", len(splitsText))
				for i, split := range splitsText {
					if i%2 == 0 {
						continue
					}
					split.SetText("0:0")
				}
				runTimer.SetText("0:0")
				select {
				case done <- struct{}{}:
				default:
				}
				needsReset = false
			}
		}
	}()
	w.Resize(fyne.Size{Height: 150, Width: 380})
	w.ShowAndRun()

}
