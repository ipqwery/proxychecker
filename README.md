# Proxy Checker Application

A GUI-based application for managing and checking proxy servers, built using the [Fyne](https://fyne.io/) framework.


![Proxy Checker Screenshot](/assets/proxychecker.png)

## Features
- Runs on all OS
- Load proxies from a file.
- Sort and filter proxies by type, address, latency, and status.
- Check proxy availability and latency using multiple threads.
- Save good or bad proxies to separate files.

## Prerequisites
- Go 1.19 or later.
- [Fyne](https://fyne.io/) framework installed.

## Installation
1. Clone the repository:
   ```sh
   git clone https://github.com/ipqwery/proxychecker.git
   cd proxychecker
   ```
2. Install dependencies:
   ```sh
   go mod tidy
   ```

## Running the Application
1. Run the application:
   ```sh
   go run .
   ```

2. The application will open a graphical user interface for managing proxies.

## Usage
- **Load Proxies**: Click "Load Proxies" to import a list of proxy addresses from a file.
- **Start Checking**: Set the timeout and thread count, then click "Start Checking" to test proxy availability.
- **Stop Checking**: Use "Stop Checking" to interrupt the current checking process.
- **Save Proxies**: Save good or bad proxies using the "Save Good Proxies" or "Save Bad Proxies" buttons.

## File Format
- Input files should contain one proxy address per line in the format `IP:Port`.

## Notes
- Adjust thread count and timeout settings for better performance depending on the number of proxies.
- Ensure a stable internet connection for accurate results.

## License
This project is licensed under the MIT License. See `LICENSE` for details.
