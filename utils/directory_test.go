package utils

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestGetFilesWithinDirectories(t *testing.T) {
	tmpA := t.TempDir()
	tmpB := t.TempDir()

	// Prepare all tested files
	allFiles := []string{filepath.Join(tmpA, "fileA"), filepath.Join(tmpB, "fileB"), filepath.Join(tmpB, "File.suf")}
	for _, fName := range allFiles {
		f, err := os.Create(fName)
		if err != nil {
			t.Fatal(err)
		}
		err = f.Close()
		if err != nil {
			t.Fatal(err)
		}
	}

	tests := []struct {
		name   string
		want   []string
		suffix string
	}{
		{
			name: "No-suffix_Expect-all",
			want: allFiles,
		},
		{
			name:   "Has-suffix_Expect-one",
			want:   []string{filepath.Join(tmpB, "File.suf")},
			suffix: "suf",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			got, err := GetFilesWithinDirectories(test.suffix, []string{tmpA, tmpB})
			if err != nil {
				t.Fatal(err)
			}

			slices.Sort(got)
			slices.Sort(test.want)
			if slices.Compare(got, test.want) != 0 {
				t.Errorf("unexpected files\n got: %v\nwant: %v", got, test.want)
			}
		})
	}
}

func TestGetFilesWithinDirectories_IgnoresLostFoundErr(t *testing.T) {
	tmp := t.TempDir()
	lostFound := filepath.Join(tmp, "lost+found")
	err := os.Mkdir(lostFound, 0777)
	if err != nil {
		t.Fatalf("cannot create lost+found dir: %v", err)
	}

	// remove ability to read from dir
	err = os.Chmod(lostFound, 0000)
	if err != nil {
		t.Fatalf("cannot chmod: %v", err)
	}

	_, err = GetFilesWithinDirectories("", []string{tmp})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}
