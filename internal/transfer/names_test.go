package transfer

import "testing"

func TestSafeName(t *testing.T) {
	got, err := SafeName(`C:\Users\a\photo.jpg`)
	if err != nil || got != "photo.jpg" {
		t.Fatalf("got %q %v", got, err)
	}
	got, err = SafeName("a/b.jpg")
	if err != nil || got != "b.jpg" {
		t.Fatalf("nested name: got %q %v", got, err)
	}
	for _, bad := range []string{"", ".", "..", "folder/..", ".gads-file-transfer.json", "has\x00null.jpg"} {
		if _, err := SafeName(bad); err == nil {
			t.Fatalf("accepted %q", bad)
		}
	}
}

func TestClassify(t *testing.T) {
	if Classify("Trip.JPEG") != KindMedia {
		t.Fatal("jpeg")
	}
	if Classify("clip.MP4") != KindMedia {
		t.Fatal("mp4")
	}
	if Classify("notes.pdf") != KindFile {
		t.Fatal("pdf")
	}
}
