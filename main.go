package main

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/olekukonko/tablewriter"
)

func main() {
	// CLI args
	f, closeFile, err := openProcessingFile(os.Args...)
	if err != nil {
		log.Fatal(err)
	}
	defer closeFile()

	// Load and parse processes
	processes, err := loadProcesses(f)
	if err != nil {
		log.Fatal(err)
	}

	// First-come, first-serve scheduling
	FCFSSchedule(os.Stdout, "First-come, first-serve", processes)

	// Shortest job first
	SJFSchedule(os.Stdout, "Shortest-job-first", processes)

	// Shortest job first, priority
	SJFPrioritySchedule(os.Stdout, "Priority", processes)

	// Round robin
	RRSchedule(os.Stdout, "Round-robin", processes)
}

func openProcessingFile(args ...string) (*os.File, func(), error) {
	if len(args) != 2 {
		return nil, nil, fmt.Errorf("%w: must give a scheduling file to process", ErrInvalidArgs)
	}
	// Read in CSV process CSV file
	f, err := os.Open(args[1])
	if err != nil {
		return nil, nil, fmt.Errorf("%v: error opening scheduling file", err)
	}
	closeFn := func() {
		if err := f.Close(); err != nil {
			log.Fatalf("%v: error closing scheduling file", err)
		}
	}

	return f, closeFn, nil
}

type (
	Process struct {
		ProcessID     int64
		ArrivalTime   int64
		BurstDuration int64
		Priority      int64
	}
	TimeSlice struct {
		PID   int64
		Start int64
		Stop  int64
	}
)

// Sorting helper functions
func sortBurstDuration(processes []Process) []Process {
	sort.Slice(processes, func(i, j int) bool {
		return processes[i].BurstDuration < processes[j].BurstDuration
	})

	return processes
}

func sortPriority(processes []Process) []Process {
	sort.Slice(processes, func(i, j int) bool {
		return processes[i].Priority < processes[j].Priority
	})

	return processes
}

func sortArrivalTime(processes []Process) []Process {
	sort.Slice(processes, func(i, j int) bool {
		return processes[i].ArrivalTime < processes[j].ArrivalTime
	})

	return processes
}

//region Schedulers

// FCFSSchedule outputs a schedule of processes in a GANTT chart and a table of timing given:
// • an output writer
// • a title for the chart
// • a slice of processes
func FCFSSchedule(w io.Writer, title string, processes []Process) {
	var (
		serviceTime     int64
		totalWait       float64
		totalTurnaround float64
		lastCompletion  float64
		waitingTime     int64
		schedule        = make([][]string, len(processes))
		gantt           = make([]TimeSlice, 0)
	)
	for i := range processes {
		if processes[i].ArrivalTime > 0 {
			waitingTime = serviceTime - processes[i].ArrivalTime
		}
		totalWait += float64(waitingTime)

		start := waitingTime + processes[i].ArrivalTime

		turnaround := processes[i].BurstDuration + waitingTime
		totalTurnaround += float64(turnaround)

		completion := processes[i].BurstDuration + processes[i].ArrivalTime + waitingTime
		lastCompletion = float64(completion)

		schedule[i] = []string{
			fmt.Sprint(processes[i].ProcessID),
			fmt.Sprint(processes[i].Priority),
			fmt.Sprint(processes[i].BurstDuration),
			fmt.Sprint(processes[i].ArrivalTime),
			fmt.Sprint(waitingTime),
			fmt.Sprint(turnaround),
			fmt.Sprint(completion),
		}
		serviceTime += processes[i].BurstDuration

		gantt = append(gantt, TimeSlice{
			PID:   processes[i].ProcessID,
			Start: start,
			Stop:  serviceTime,
		})
	}

	count := float64(len(processes))
	aveWait := totalWait / count
	aveTurnaround := totalTurnaround / count
	aveThroughput := count / lastCompletion

	outputTitle(w, title)
	outputGantt(w, gantt)
	outputSchedule(w, schedule, aveWait, aveTurnaround, aveThroughput)
}

// Shortest job first, priority
func SJFPrioritySchedule(w io.Writer, title string, processes []Process) {
	var (
		serviceTime     int64
		totalWait       float64
		totalTurnaround float64
		lastCompletion  float64
		waitingTime     int64
		schedule        = make([][]string, len(processes))
		gantt           = make([]TimeSlice, 0)
	)

	processes = sortPriority(processes)

	for i := range processes {
		// Calculate waiting time
		if i > 0 {
			waitingTime += processes[i-1].BurstDuration
		}

		// Add to total waiting time
		totalWait += float64(waitingTime)

		// Calculate start time
		start := waitingTime

		// Calculate turnaround time
		turnaround := processes[i].BurstDuration + waitingTime
		totalTurnaround += float64(turnaround)

		// Calculate completion time
		completion := processes[i].BurstDuration + waitingTime
		lastCompletion = float64(completion)

		// Add to schedule
		schedule[i] = []string{
			fmt.Sprint(processes[i].ProcessID),
			fmt.Sprint(processes[i].Priority),
			fmt.Sprint(processes[i].BurstDuration),
			fmt.Sprint(processes[i].ArrivalTime),
			fmt.Sprint(waitingTime),
			fmt.Sprint(turnaround),
			fmt.Sprint(completion),
		}

		// Add to service time
		serviceTime += processes[i].BurstDuration

		// Add to GANTT chart
		gantt = append(gantt, TimeSlice{
			PID:   processes[i].ProcessID,
			Start: start,
			Stop:  serviceTime,
		})
	}

	count := float64(len(processes))
	aveWait := totalWait / count
	aveTurnaround := totalTurnaround / count
	aveThroughput := count / lastCompletion

	outputTitle(w, title)
	outputGantt(w, gantt)
	outputSchedule(w, schedule, aveWait, aveTurnaround, aveThroughput)
}

// Shortest job first
func SJFSchedule(w io.Writer, title string, processes []Process) {
	var (
		serviceTime     int64
		totalWait       float64
		totalTurnaround float64
		lastCompletion  float64
		waitingTime     int64
		schedule        = make([][]string, len(processes))
		gantt           = make([]TimeSlice, 0)
	)

	processes = sortBurstDuration(processes)

	for i := range processes {
		// Calculate waiting time
		if i > 0 {
			waitingTime += processes[i-1].BurstDuration
		}

		// Add to total waiting time
		totalWait += float64(waitingTime)

		// Calculate start time
		start := waitingTime

		// Calculate turnaround time
		turnaround := processes[i].BurstDuration + waitingTime
		totalTurnaround += float64(turnaround)

		// Calculate completion time
		completion := processes[i].BurstDuration + waitingTime
		lastCompletion = float64(completion)

		// Add to schedule
		schedule[i] = []string{
			fmt.Sprint(processes[i].ProcessID),
			fmt.Sprint(processes[i].Priority),
			fmt.Sprint(processes[i].BurstDuration),
			fmt.Sprint(processes[i].ArrivalTime),
			fmt.Sprint(waitingTime),
			fmt.Sprint(turnaround),
			fmt.Sprint(completion),
		}

		// Add to service time
		serviceTime += processes[i].BurstDuration

		// Add to GANTT chart
		gantt = append(gantt, TimeSlice{
			PID:   processes[i].ProcessID,
			Start: start,
			Stop:  serviceTime,
		})
	}

	count := float64(len(processes))
	aveWait := totalWait / count
	aveTurnaround := totalTurnaround / count
	aveThroughput := count / lastCompletion

	outputTitle(w, title)
	outputGantt(w, gantt)
	outputSchedule(w, schedule, aveWait, aveTurnaround, aveThroughput)
}

// Round robin
func RRSchedule(w io.Writer, title string, processes []Process) {
	var (
		serviceTime     int64
		totalWait       float64
		totalTurnaround float64
		lastCompletion  float64
		waitingTime     int64
		schedule        = make([][]string, len(processes))
		gantt           = make([]TimeSlice, 0)
		// Max amount of time that each process can execute before moving onto next process
		quantumTime int64
		// Processes completed check
		procsCompleted int64
		// Arrays of ints to keep track of the time left and wait times for each process
		timeLeft  []int64
		waitTimes []int64
		// Count to keep track of total number of time units that have elapsed
		countTimeUnits float64
		// Sum of wait times
		sumWaitTimes float64
		// Minimun calculation var to ensure that no process runs longer than quantumTime
		min int64
	)

	processes = sortArrivalTime(processes)

	quantumTime = 5
	countTimeUnits = 0.0
	sumWaitTimes = 0.0

	// Create timeLeft array with the burst duration of each process
	for i := range processes {
		timeLeft = append(timeLeft, processes[i].BurstDuration)
		waitTimes = append(waitTimes, 0)
	}

	for procsCompleted < int64(len(processes)) {
		// Iterate over processes
		for i := range processes {

			// If the time left at i is less than zero, continue
			if timeLeft[i] <= 0 {
				continue
			}

			min = int64(math.Min(float64(quantumTime), float64(timeLeft[i])))

			// Calculate and update the waiting time for a process
			if countTimeUnits > 0 {
				waitingTime += min
				waitTimes[i] += min
			}

			countTimeUnits++

			// Add to total waiting time
			totalWait += float64(waitingTime)

			// Calculate start time
			start := waitingTime

			// Calculate turnaround time
			turnaround := min + waitingTime
			totalTurnaround += float64(turnaround)

			// Calculate completion time
			completion := min + waitingTime
			lastCompletion = float64(completion)

			// Append to schedule
			schedule = append(schedule, []string{
				fmt.Sprint(processes[i].ProcessID),
				fmt.Sprint(processes[i].Priority),
				fmt.Sprint(processes[i].BurstDuration),
				fmt.Sprint(processes[i].ArrivalTime),
				fmt.Sprint(waitingTime),
				fmt.Sprint(turnaround),
				fmt.Sprint(completion),
			})

			// Add to service time
			serviceTime = start + min

			// Add to GANTT chart
			gantt = append(gantt, TimeSlice{
				PID:   processes[i].ProcessID,
				Start: start,
				Stop:  serviceTime,
			})

			// Subtract the min of the quantumTime and the time left at i from the time left at i
			timeLeft[i] -= min

			// If the time left at i is less than or equal to zero, increment procsCompleted
			if timeLeft[i] <= 0 {
				procsCompleted++
			}

		}

	}

	// Get the sum of the wait times
	for i := range waitTimes {
		sumWaitTimes += float64(waitTimes[i])
	}

	aveWait := sumWaitTimes / float64(len(processes))
	aveTurnaround := totalTurnaround / countTimeUnits
	aveThroughput := countTimeUnits / lastCompletion

	outputTitle(w, title)
	outputGantt(w, gantt)
	outputSchedule(w, schedule, aveWait, aveTurnaround, aveThroughput)
}

//endregion

//region Output helpers

func outputTitle(w io.Writer, title string) {
	_, _ = fmt.Fprintln(w, strings.Repeat("-", len(title)*2))
	_, _ = fmt.Fprintln(w, strings.Repeat(" ", len(title)/2), title)
	_, _ = fmt.Fprintln(w, strings.Repeat("-", len(title)*2))
}

func outputGantt(w io.Writer, gantt []TimeSlice) {
	_, _ = fmt.Fprintln(w, "Gantt schedule")
	_, _ = fmt.Fprint(w, "|")
	for i := range gantt {
		pid := fmt.Sprint(gantt[i].PID)
		padding := strings.Repeat(" ", (8-len(pid))/2)
		_, _ = fmt.Fprint(w, padding, pid, padding, "|")
	}
	_, _ = fmt.Fprintln(w)
	for i := range gantt {
		_, _ = fmt.Fprint(w, fmt.Sprint(gantt[i].Start), "\t")
		if len(gantt)-1 == i {
			_, _ = fmt.Fprint(w, fmt.Sprint(gantt[i].Stop))
		}
	}
	_, _ = fmt.Fprintf(w, "\n\n")
}

func outputSchedule(w io.Writer, rows [][]string, wait, turnaround, throughput float64) {
	_, _ = fmt.Fprintln(w, "Schedule table")
	table := tablewriter.NewWriter(w)
	table.SetHeader([]string{"ID", "Priority", "Burst", "Arrival", "Wait", "Turnaround", "Exit"})
	table.AppendBulk(rows)
	table.SetFooter([]string{"", "", "", "",
		fmt.Sprintf("Average\n%.2f", wait),
		fmt.Sprintf("Average\n%.2f", turnaround),
		fmt.Sprintf("Throughput\n%.2f/t", throughput)})
	table.Render()
}

//endregion

//region Loading processes.

var ErrInvalidArgs = errors.New("invalid args")

func loadProcesses(r io.Reader) ([]Process, error) {
	rows, err := csv.NewReader(r).ReadAll()
	if err != nil {
		return nil, fmt.Errorf("%w: reading CSV", err)
	}

	processes := make([]Process, len(rows))
	for i := range rows {
		processes[i].ProcessID = mustStrToInt(rows[i][0])
		processes[i].BurstDuration = mustStrToInt(rows[i][1])
		processes[i].ArrivalTime = mustStrToInt(rows[i][2])
		if len(rows[i]) == 4 {
			processes[i].Priority = mustStrToInt(rows[i][3])
		}
	}

	return processes, nil
}

func mustStrToInt(s string) int64 {
	i, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	return i
}

//endregion
