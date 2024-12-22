package humanreadable

import "fmt"

type BiBytes int64

func (n BiBytes) String() string {
	return sizeStringify(int64(n), 1024, []string{"B", "KiB", "MiB", "GiB", "TiB", "PiB", "EiB"})
}

type Bytes int64

func (n Bytes) String() string {
	return sizeStringify(int64(n), 1000, []string{"B", "KB", "MB", "GB", "TB", "PB", "EB"})
}

func sizeStringify(n int64, unit int, suffixes []string) string {
	fsize := float64(n)

	for _, suffix := range suffixes {
		if fsize < float64(unit) {
			return fmt.Sprintf("%.1f%s", fsize, suffix)
		}

		fsize /= float64(unit)
	}

	return fmt.Sprintf("%.1fEiB", fsize)
}
