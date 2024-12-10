package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

type ProxyEntry struct {
	Address string
	Type    string
	Latency time.Duration
	Status  string
}

type ProxyList struct {
	entries []ProxyEntry
	sortCol int
	asc     bool
}

func (pl *ProxyList) Len() int {
	return len(pl.entries)
}

func (pl *ProxyList) Less(i, j int) bool {
	switch pl.sortCol {
	case 0: // Type
		if pl.asc {
			return pl.entries[i].Type < pl.entries[j].Type
		}
		return pl.entries[i].Type > pl.entries[j].Type
	case 1: // Address
		if pl.asc {
			return pl.entries[i].Address < pl.entries[j].Address
		}
		return pl.entries[i].Address > pl.entries[j].Address
	case 2: // Latency
		if pl.asc {
			return pl.entries[i].Latency < pl.entries[j].Latency
		}
		return pl.entries[i].Latency > pl.entries[j].Latency
	case 3: // Status
		if pl.asc {
			return pl.entries[i].Status < pl.entries[j].Status
		}
		return pl.entries[i].Status > pl.entries[j].Status
	default:
		return false
	}
}

func (pl *ProxyList) Swap(i, j int) {
	pl.entries[i], pl.entries[j] = pl.entries[j], pl.entries[i]
}

type CellWrapper struct {
	widget.BaseWidget
	content fyne.CanvasObject
	proxy   ProxyEntry
	window  fyne.Window
}

func NewCellWrapper(content fyne.CanvasObject, proxy ProxyEntry, window fyne.Window) *CellWrapper {
	c := &CellWrapper{
		content: content,
		proxy:   proxy,
		window:  window,
	}
	c.ExtendBaseWidget(c)
	return c
}

func (c *CellWrapper) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(c.content)
}

func (c *CellWrapper) Tapped(*fyne.PointEvent) {
	// No action on left-click
}

func (c *CellWrapper) TappedSecondary(*fyne.PointEvent) {
	// On right-click, copy the proxy
	proxyURL := fmt.Sprintf("%s://%s", strings.ToLower(c.proxy.Type), c.proxy.Address)
	c.window.Clipboard().SetContent(proxyURL)
	dialog.ShowInformation("Copied to Clipboard", fmt.Sprintf("Proxy URL: %s\nhas been copied to the clipboard.", proxyURL), c.window)
}

func main() {
	myApp := app.New()
	myWindow := myApp.NewWindow("Proxy Checker")

	// Fixed window size and prevent resizing
	myWindow.Resize(fyne.NewSize(800, 600))
	myWindow.SetFixedSize(true)

	// Data storage
	proxyList := &ProxyList{
		entries: []ProxyEntry{},
		sortCol: 1,
		asc:     true,
	}
	goodProxies := []ProxyEntry{}
	badProxies := []ProxyEntry{}

	// Thread count
	threadCount := 50
	threadEntry := widget.NewEntry()
	threadEntry.SetPlaceHolder("Threads")
	threadEntry.SetText(fmt.Sprintf("%d", threadCount))

	// Timeout input
	timeoutEntry := widget.NewEntry()
	timeoutEntry.SetPlaceHolder("Timeout (seconds)")
	timeoutEntry.SetText("5")

	// Proxy type selection
	proxyType := widget.NewSelect([]string{"SOCKS5", "SOCKS4", "HTTP"}, func(value string) {})
	proxyType.SetSelected("HTTP")

	// Sorting variables
	var table *widget.Table
	headers := []string{"Proxy Type", "Address", "Latency", "Status"}

	// Table for proxies
	table = widget.NewTable(
		func() (int, int) {
			return len(proxyList.entries) + 1, 4
		},
		func() fyne.CanvasObject {
			// Each cell is a container
			return container.NewMax()
		},
		func(id widget.TableCellID, cell fyne.CanvasObject) {
			c := cell.(*fyne.Container)
			c.Objects = nil // Clear previous objects

			if id.Row == 0 {
				// Table headers
				header := widget.NewButton(headers[id.Col], func() {
					if proxyList.sortCol == id.Col {
						proxyList.asc = !proxyList.asc
					} else {
						proxyList.sortCol = id.Col
						proxyList.asc = true
					}
					sort.Sort(proxyList)
					table.Refresh()
				})
				header.Alignment = widget.ButtonAlignCenter
				header.Importance = widget.LowImportance
				c.Add(header)
			} else {
				idx := id.Row - 1
				if idx >= len(proxyList.entries) {
					return
				}
				proxy := proxyList.entries[idx]
				var content fyne.CanvasObject
				switch id.Col {
				case 0:
					label := widget.NewLabel(proxy.Type)
					label.Alignment = fyne.TextAlignCenter
					content = label
				case 1:
					label := widget.NewLabel(proxy.Address)
					label.Alignment = fyne.TextAlignCenter
					content = label
				case 2:
					label := widget.NewLabel(fmt.Sprintf("%v", proxy.Latency))
					label.Alignment = fyne.TextAlignCenter
					content = label
				case 3:
					// Status as emoji
					var statusEmoji string
					if proxy.Status == "Success" {
						statusEmoji = "✅"
					} else {
						statusEmoji = "❌"
					}
					label := widget.NewLabel(statusEmoji)
					label.Alignment = fyne.TextAlignCenter
					content = label
				}
				// Wrap content with CellWrapper
				wrapper := NewCellWrapper(content, proxy, myWindow)
				c.Add(wrapper)
			}
		},
	)
	table.SetColumnWidth(0, 100) // Proxy Type
	table.SetColumnWidth(1, 250) // Address
	table.SetColumnWidth(2, 100) // Latency
	table.SetColumnWidth(3, 50)  // Status

	// Set the header row height
	table.SetRowHeight(0, 40) // Header row

	// Function to update row heights
	updateRowHeights := func() {
		for i := 1; i <= len(proxyList.entries); i++ {
			table.SetRowHeight(i, 30) // Set data row height
		}
	}

	// Wrap the table in a scroll container
	tableScroll := container.NewVScroll(table)

	// Load proxies button
	loadButton := widget.NewButton("Load Proxies", func() {
		dialog.ShowFileOpen(func(reader fyne.URIReadCloser, err error) {
			if err != nil || reader == nil {
				return
			}
			defer reader.Close()

			proxyList.entries = []ProxyEntry{} // Clear current proxies
			scanner := bufio.NewScanner(reader)
			for scanner.Scan() {
				line := strings.TrimSpace(scanner.Text())
				if line != "" {
					proxyList.entries = append(proxyList.entries, ProxyEntry{Address: line, Type: proxyType.Selected, Status: "Pending"})
				}
			}
			updateRowHeights()
			table.Refresh()
		}, myWindow)
	})

	// Start and stop buttons
	var wg sync.WaitGroup
	var stopChecking bool
	startButton := widget.NewButton("Start Checking", func() {
		stopChecking = false
		timeout, err := time.ParseDuration(timeoutEntry.Text + "s")
		if err != nil {
			dialog.ShowError(fmt.Errorf("invalid timeout value"), myWindow)
			return
		}

		threads, err := parseInt(threadEntry.Text, 50)
		if err != nil || threads <= 0 {
			dialog.ShowError(fmt.Errorf("invalid thread count"), myWindow)
			return
		}

		sem := make(chan struct{}, threads) // Semaphore for thread control
		goodProxies = []ProxyEntry{}
		badProxies = []ProxyEntry{}

		for i := range proxyList.entries {
			if stopChecking {
				break
			}
			wg.Add(1)
			sem <- struct{}{}
			go func(i int) {
				defer wg.Done()
				defer func() { <-sem }()

				start := time.Now()
				status := checkProxy(proxyList.entries[i].Address, proxyList.entries[i].Type, timeout)
				proxyList.entries[i].Latency = time.Since(start)
				proxyList.entries[i].Status = status

				if status == "Success" {
					goodProxies = append(goodProxies, proxyList.entries[i])
				} else {
					badProxies = append(badProxies, proxyList.entries[i])
				}
				table.Refresh()
			}(i)
		}
		wg.Wait()
		close(sem)
	})
	stopButton := widget.NewButton("Stop Checking", func() {
		stopChecking = true
	})

	// Save good/bad proxies
	saveGoodButton := widget.NewButton("Save Good Proxies", func() {
		dialog.ShowFileSave(func(writer fyne.URIWriteCloser, err error) {
			if err != nil || writer == nil {
				return
			}
			defer writer.Close()

			for _, proxy := range goodProxies {
				writer.Write([]byte(fmt.Sprintf("%s\n", proxy.Address)))
			}
		}, myWindow)
	})

	saveBadButton := widget.NewButton("Save Bad Proxies", func() {
		dialog.ShowFileSave(func(writer fyne.URIWriteCloser, err error) {
			if err != nil || writer == nil {
				return
			}
			defer writer.Close()

			for _, proxy := range badProxies {
				writer.Write([]byte(fmt.Sprintf("%s\n", proxy.Address)))
			}
		}, myWindow)
	})

	// Layout
	controlPanel := container.NewVBox(
		widget.NewLabel("Proxy Type:"),
		proxyType,
		widget.NewLabel("Timeout (seconds):"),
		timeoutEntry,
		widget.NewLabel("Threads:"),
		threadEntry,
		loadButton,
		startButton,
		stopButton,
		saveGoodButton,
		saveBadButton,
	)

	// Use a Split container to allocate space between control panel and table
	split := container.NewHSplit(controlPanel, tableScroll)
	split.Offset = 0.2 // Adjusted to make control panel 1/5th and table 4/5ths

	myWindow.SetContent(split)
	myWindow.ShowAndRun()
}

func checkProxy(proxyAddr, proxyType string, timeout time.Duration) string {
	proxyURL, err := url.Parse(fmt.Sprintf("%s://%s", strings.ToLower(proxyType), proxyAddr))
	if err != nil {
		return "Invalid URL"
	}

	transport := &http.Transport{
		Proxy: http.ProxyURL(proxyURL),
		DialContext: (&net.Dialer{
			Timeout: timeout,
		}).DialContext,
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   timeout,
	}

	req, err := http.NewRequest("GET", "https://api.ipquery.io/?format=json&filter=location", nil)
	if err != nil {
		return "Failed"
	}

	resp, err := client.Do(req)
	if err != nil {
		return "Failed"
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		// Optionally, read the body to ensure it's valid JSON
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return "Failed"
		}
		if len(body) == 0 {
			return "Failed"
		}
		// You could parse the JSON if needed, but for our purposes, we can assume success
		return "Success"
	}
	return "Failed"
}

func parseInt(value string, defaultValue int) (int, error) {
	if value == "" {
		return defaultValue, nil
	}
	var result int
	_, err := fmt.Sscanf(value, "%d", &result)
	if err != nil {
		return 0, err
	}
	return result, nil
}
