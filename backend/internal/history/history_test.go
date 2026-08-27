package history

import (
	"fmt"
	"reflect"
	"sync"
	"testing"
)

func TestImageHistoryAddKeepsRecentSupportedImages(t *testing.T) {
	history := newImageHistory()
	for _, image := range []string{
		"https://images.example.com/one.png",
		"https://images.example.com/two.png",
		"https://images.example.com/three.png",
		"https://images.example.com/four.png",
		"https://images.example.com/five.png",
		"https://images.example.com/six.png",
		"data:image/png;base64,aGVsbG8=",
	} {
		history.Add(image)
	}
	want := []string{
		"data:image/png;base64,aGVsbG8=",
		"https://images.example.com/six.png",
		"https://images.example.com/five.png",
		"https://images.example.com/four.png",
		"https://images.example.com/three.png",
	}
	if got := history.List(); !reflect.DeepEqual(got, want) {
		t.Fatalf("List() = %#v, want %#v", got, want)
	}
	for _, image := range []string{"data:image/png;base64,abc", "data:image/png,not-base64", "http://images.example.com/image.png", "not a URL", "https://"} {
		history.Add(image)
	}
	if got := history.List(); !reflect.DeepEqual(got, want) {
		t.Fatalf("List() after invalid Add() = %#v, want %#v", got, want)
	}
}

func TestImageHistoryDuplicateMovesToFront(t *testing.T) {
	history := newImageHistory()
	history.Add("https://images.example.com/one.png")
	history.Add("https://images.example.com/two.png")
	history.Add("https://images.example.com/one.png")
	if got, want := history.List(), []string{"https://images.example.com/one.png", "https://images.example.com/two.png"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("List() = %#v, want %#v", got, want)
	}
}

func TestImageHistoryRemove(t *testing.T) {
	history := newImageHistory()
	history.Add("https://images.example.com/one.png")
	history.Add("https://images.example.com/two.png")
	history.Remove("https://images.example.com/one.png")
	want := []string{"https://images.example.com/two.png"}
	if got := history.List(); !reflect.DeepEqual(got, want) {
		t.Fatalf("List() = %#v, want %#v", got, want)
	}
	history.Remove("http://images.example.com/two.png")
	if got := history.List(); !reflect.DeepEqual(got, want) {
		t.Fatalf("invalid Remove() changed List() to %#v", got)
	}
}

func TestImageHistoryConcurrentAccess(t *testing.T) {
	history := newImageHistory()
	var waitGroup sync.WaitGroup
	for worker := 0; worker < 8; worker++ {
		worker := worker
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			for index := 0; index < 100; index++ {
				image := fmt.Sprintf("https://images.example.com/%d/%d.png", worker, index)
				history.Add(image)
				_ = history.List()
				history.Remove(image)
			}
		}()
	}
	waitGroup.Wait()
	if got := len(history.List()); got > maxImageHistory {
		t.Fatalf("List() returned %d items, want at most %d", got, maxImageHistory)
	}
}
