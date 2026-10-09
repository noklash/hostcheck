package main

import (
	"fmt"
	"io"
	"net"
	"sort"
	"strings"
	"time"

	"github.com/noklash/hostcheck/internal/health"
	"github.com/noklash/hostcheck/internal/host"
	"github.com/noklash/hostcheck/internal/memory"
	"github.com/noklash/hostcheck/internal/network"
)

// writef writes formatted output and returns only the error.
func writef(w io.Writer, format string, args ...any) error {
	_, err := fmt.Fprintf(w, format, args...)
	return err
}

// writeln writes a line and returns only the error.
func writeln(w io.Writer, args ...any) error {
	_, err := fmt.Fprintln(w, args...)
	return err
}

func printResult(
	w io.Writer,
	snapshot host.Snapshot,
	result health.Result,
) error {
	if err := writeln(w, "hostcheck"); err != nil {
		return err
	}

	if err := writef(
		w,
		"observed_at: %s\n",
		snapshot.ObservedAt.Format(time.RFC3339),
	); err != nil {
		return err
	}

	status := string(result.Status)
	if status == "" {
		status = "unavailable"
	}

	if err := writef(w, "status: %s\n", status); err != nil {
		return err
	}

	if err := writef(w, "coverage: %s\n\n", result.Coverage); err != nil {
		return err
	}

	if err := printCollection(w, snapshot); err != nil {
		return err
	}
	if err := printCPU(w, snapshot); err != nil {
		return err
	}
	if err := printMemory(w, snapshot); err != nil {
		return err
	}
	if err := printFilesystem(w, snapshot); err != nil {
		return err
	}
	if err := printProcesses(w, snapshot); err != nil {
		return err
	}
	if err := printNetwork(w, snapshot); err != nil {
		return err
	}

	return printAssessments(w, result)
}

func printCollection(w io.Writer, snapshot host.Snapshot) error {
	if err := writeln(w, "collection"); err != nil {
		return err
	}

	subsystems := []struct {
		name      string
		available bool
	}{
		{name: "cpu", available: snapshot.CPU != nil},
		{name: "memory", available: snapshot.Memory != nil},
		{name: "filesystem", available: snapshot.Filesystem != nil},
		{name: "process", available: snapshot.Processes != nil},
		{name: "network", available: snapshot.Network != nil},
	}

	for _, subsystem := range subsystems {
		if subsystem.available {
			if err := writef(w, "  %s: collected\n", subsystem.name); err != nil {
				return err
			}
			continue
		}

		err := collectionError(snapshot, subsystem.name)
		if err != nil {
			if writeErr := writef(
				w,
				"  %s: failed: %v\n",
				subsystem.name,
				err,
			); writeErr != nil {
				return writeErr
			}
			continue
		}

		if err := writef(w, "  %s: unavailable\n", subsystem.name); err != nil {
			return err
		}
	}

	return writeln(w)
}

func printCPU(w io.Writer, snapshot host.Snapshot) error {
	if err := writeln(w, "cpu"); err != nil {
		return err
	}

	if snapshot.CPU == nil {
		if err := printSectionError(w, snapshot, "cpu"); err != nil {
			return err
		}
		return writeln(w)
	}

	cpu := snapshot.CPU

	fields := []struct {
		name  string
		value uint64
	}{
		{name: "user_ticks", value: cpu.User},
		{name: "nice_ticks", value: cpu.Nice},
		{name: "system_ticks", value: cpu.System},
		{name: "idle_ticks", value: cpu.Idle},
		{name: "iowait_ticks", value: cpu.IOWait},
		{name: "irq_ticks", value: cpu.IRQ},
		{name: "softirq_ticks", value: cpu.SoftIRQ},
		{name: "steal_ticks", value: cpu.Steal},
		{name: "guest_ticks", value: cpu.Guest},
		{name: "guest_nice_ticks", value: cpu.GuestNice},
	}

	for _, field := range fields {
		if err := writef(
			w,
			"  %s: %d\n",
			field.name,
			field.value,
		); err != nil {
			return err
		}
	}

	if err := writeln(
		w,
		"  utilization: not evaluated (requires a second sample)",
	); err != nil {
		return err
	}

	return writeln(w)
}

func printMemory(w io.Writer, snapshot host.Snapshot) error {
	if err := writeln(w, "memory"); err != nil {
		return err
	}

	if snapshot.Memory == nil {
		if err := printSectionError(w, snapshot, "memory"); err != nil {
			return err
		}
		return writeln(w)
	}

	mem := *snapshot.Memory

	fields := []struct {
		name  string
		value uint64
	}{
		{name: "total_kb", value: mem.Total},
		{name: "available_kb", value: mem.Available},
		{name: "free_kb", value: mem.Free},
		{name: "buffers_kb", value: mem.Buffers},
		{name: "cached_kb", value: mem.Cached},
		{name: "swap_total_kb", value: mem.SwapTotal},
		{name: "swap_free_kb", value: mem.SwapFree},
	}

	for _, field := range fields {
		if err := writef(
			w,
			"  %s: %d\n",
			field.name,
			field.value,
		); err != nil {
			return err
		}
	}

	availablePercent, err := memory.AvailablePercent(mem)
	if err != nil {
		if writeErr := writef(
			w,
			"  available_percent: unavailable (%v)\n",
			err,
		); writeErr != nil {
			return writeErr
		}
	} else {
		if err := writef(
			w,
			"  available_percent: %.2f\n",
			availablePercent,
		); err != nil {
			return err
		}
	}

	return writeln(w)
}

func printFilesystem(w io.Writer, snapshot host.Snapshot) error {
	if err := writeln(w, "filesystem"); err != nil {
		return err
	}

	if snapshot.Filesystem == nil {
		if err := printSectionError(w, snapshot, "filesystem"); err != nil {
			return err
		}
		return writeln(w)
	}

	fs := snapshot.Filesystem

	fields := []struct {
		name  string
		value any
	}{
		{name: "path", value: fs.Path},
		{name: "block_size", value: fs.BlockSize},
		{name: "blocks_total", value: fs.BlocksTotal},
		{name: "blocks_free", value: fs.BlocksFree},
		{name: "blocks_available", value: fs.BlocksAvailable},
		{name: "inodes_total", value: fs.InodesTotal},
		{name: "inodes_free", value: fs.InodesFree},
	}

	for _, field := range fields {
		if err := writef(
			w,
			"  %s: %v\n",
			field.name,
			field.value,
		); err != nil {
			return err
		}
	}

	usedBlocks, err := fs.UsedBlocks()
	if err != nil {
		if writeErr := writef(
			w,
			"  used_blocks: unavailable (%v)\n",
			err,
		); writeErr != nil {
			return writeErr
		}
	} else {
		if err := writef(w, "  used_blocks: %d\n", usedBlocks); err != nil {
			return err
		}
	}

	usedInodes, err := fs.UsedInodes()
	if err != nil {
		if writeErr := writef(
			w,
			"  used_inodes: unavailable (%v)\n",
			err,
		); writeErr != nil {
			return writeErr
		}
	} else {
		if err := writef(w, "  used_inodes: %d\n", usedInodes); err != nil {
			return err
		}
	}

	availableBytes, err := fs.AvailableBytes()
	if err != nil {
		if writeErr := writef(
			w,
			"  available_bytes: unavailable (%v)\n",
			err,
		); writeErr != nil {
			return writeErr
		}
	} else {
		if err := writef(
			w,
			"  available_bytes: %d\n",
			availableBytes,
		); err != nil {
			return err
		}
	}

	usedBytes, err := fs.UsedBytes()
	if err != nil {
		if writeErr := writef(
			w,
			"  used_bytes: unavailable (%v)\n",
			err,
		); writeErr != nil {
			return writeErr
		}
	} else {
		if err := writef(w, "  used_bytes: %d\n", usedBytes); err != nil {
			return err
		}
	}

	availablePercent, err := fs.AvailablePercent()
	if err != nil {
		if writeErr := writef(
			w,
			"  available_percent: unavailable (%v)\n",
			err,
		); writeErr != nil {
			return writeErr
		}
	} else {
		if err := writef(
			w,
			"  available_percent: %.2f\n",
			availablePercent,
		); err != nil {
			return err
		}
	}

	availableInodePercent, err := fs.AvailableInodePercent()
	if err != nil {
		if writeErr := writef(
			w,
			"  available_inode_percent: unavailable (%v)\n",
			err,
		); writeErr != nil {
			return writeErr
		}
	} else {
		if err := writef(
			w,
			"  available_inode_percent: %.2f\n",
			availableInodePercent,
		); err != nil {
			return err
		}
	}

	return writeln(w)
}

func printProcesses(w io.Writer, snapshot host.Snapshot) error {
	if err := writeln(w, "processes"); err != nil {
		return err
	}

	if snapshot.Processes == nil {
		if err := printSectionError(w, snapshot, "process"); err != nil {
			return err
		}
		return writeln(w)
	}

	total := len(snapshot.Processes)
	kernelThreads := 0
	states := make(map[byte]int)

	for _, proc := range snapshot.Processes {
		if proc.Kthread {
			kernelThreads++
		}
		states[proc.Stats.State]++
	}

	fields := []struct {
		name  string
		value int
	}{
		{name: "total", value: total},
		{name: "kernel_threads", value: kernelThreads},
		{name: "userspace", value: total - kernelThreads},
	}

	for _, field := range fields {
		if err := writef(
			w,
			"  %s: %d\n",
			field.name,
			field.value,
		); err != nil {
			return err
		}
	}

	if err := writeln(w, "  states:"); err != nil {
		return err
	}

	stateNames := make([]string, 0, len(states))
	for state := range states {
		stateNames = append(stateNames, string(state))
	}
	sort.Strings(stateNames)

	for _, state := range stateNames {
		if err := writef(
			w,
			"    %s: %d\n",
			state,
			states[state[0]],
		); err != nil {
			return err
		}
	}

	return writeln(w)
}

func printNetwork(w io.Writer, snapshot host.Snapshot) error {
	if err := writeln(w, "network"); err != nil {
		return err
	}

	if snapshot.Network == nil {
		if err := printSectionError(w, snapshot, "network"); err != nil {
			return err
		}
		return writeln(w)
	}

	netSnapshot := snapshot.Network

	if err := writef(
		w,
		"  observed_at: %s\n",
		netSnapshot.ObservedAt.Format(time.RFC3339),
	); err != nil {
		return err
	}

	if err := writef(
		w,
		"  interfaces: %d\n",
		len(netSnapshot.Interfaces),
	); err != nil {
		return err
	}

	for _, iface := range netSnapshot.Interfaces {
		if err := writef(w, "    %s\n", iface.Name); err != nil {
			return err
		}

		fields := []struct {
			name  string
			value any
		}{
			{name: "index", value: iface.Index},
			{name: "hardware_address", value: emptyAsDash(iface.HardwareAddr)},
			{name: "oper_state", value: iface.OperState},
			{name: "carrier", value: iface.Carrier},
			{name: "mtu", value: iface.MTU},
		}

		for _, field := range fields {
			if err := writef(
				w,
				"      %s: %v\n",
				field.name,
				field.value,
			); err != nil {
				return err
			}
		}

		if iface.SpeedMbps != nil {
			if err := writef(
				w,
				"      speed_mbps: %d\n",
				*iface.SpeedMbps,
			); err != nil {
				return err
			}
		}

		if iface.Duplex != nil {
			if err := writef(
				w,
				"      duplex: %s\n",
				*iface.Duplex,
			); err != nil {
				return err
			}
		}

		if err := writeln(w, "      addresses:"); err != nil {
			return err
		}

		if len(iface.Addresses) == 0 {
			if err := writeln(w, "        -"); err != nil {
				return err
			}
		}

		for _, address := range iface.Addresses {
			if err := writef(
				w,
				"        %s/%d scope=%d\n",
				formatIP(address.IP),
				address.PrefixLen,
				address.Scope,
			); err != nil {
				return err
			}
		}
	}

	if err := writef(
		w,
		"  routes: %d\n",
		len(netSnapshot.Routes),
	); err != nil {
		return err
	}

	interfaceNames := make(map[uint32]string, len(netSnapshot.Interfaces))
	for _, iface := range netSnapshot.Interfaces {
		interfaceNames[iface.Index] = iface.Name
	}

	for _, route := range netSnapshot.Routes {
		if err := printRoute(w, route, interfaceNames); err != nil {
			return err
		}
	}

	return writeln(w)
}

func printRoute(
	w io.Writer,
	route network.Route,
	interfaceNames map[uint32]string,
) error {
	destination := formatRoutePrefix(route.Destination, route.PrefixLen)

	// The source field is an IP address. Avoid displaying /0 when the
	// route source prefix length is zero.
	source := "-"
	if route.Source != nil {
		source = route.Source.String()
	}

	gateway := "-"
	if route.Gateway != nil {
		gateway = route.Gateway.String()
	}

	interfaceName := "-"
	if name, ok := interfaceNames[route.InterfaceIndex]; ok {
		interfaceName = name
	}

	if err := writef(w, "    %s\n", destination); err != nil {
		return err
	}

	fields := []struct {
		name  string
		value any
	}{
		{name: "family", value: route.Family},
		{name: "source", value: source},
		{name: "gateway", value: gateway},
		{name: "interface", value: interfaceName},
		{name: "interface_index", value: route.InterfaceIndex},
		{name: "priority", value: route.Priority},
		{name: "table", value: route.Table},
		{name: "protocol", value: route.Protocol},
		{name: "scope", value: route.Scope},
		{name: "type", value: route.Type},
		{name: "flags", value: route.Flags},
	}

	for _, field := range fields {
		if err := writef(
			w,
			"      %s: %v\n",
			field.name,
			field.value,
		); err != nil {
			return err
		}
	}

	if len(route.Multipath) > 0 {
		if err := writeln(w, "      multipath:"); err != nil {
			return err
		}

		for _, nextHop := range route.Multipath {
			nextHopInterface := "-"
			if name, ok := interfaceNames[nextHop.InterfaceIndex]; ok {
				nextHopInterface = name
			}

			nextHopGateway := "-"
			if nextHop.Gateway != nil {
				nextHopGateway = nextHop.Gateway.String()
			}

			if err := writef(
				w,
				"        interface=%s index=%d gateway=%s hops=%d flags=%d\n",
				nextHopInterface,
				nextHop.InterfaceIndex,
				nextHopGateway,
				nextHop.Hops,
				nextHop.Flags,
			); err != nil {
				return err
			}
		}
	}

	return nil
}

func printAssessments(w io.Writer, result health.Result) error {
	if err := writeln(w, "assessments"); err != nil {
		return err
	}

	for _, assessment := range result.Assessments {
		if assessment.Availability == health.Unassessable {
			if err := writef(
				w,
				"  [unassessable] %s\n",
				assessment.Subject,
			); err != nil {
				return err
			}
		} else {
			if err := writef(
				w,
				"  [%s] %s\n",
				assessment.Status,
				assessment.Subject,
			); err != nil {
				return err
			}
		}

		if err := writef(w, "    %s\n", assessment.Reason); err != nil {
			return err
		}

		for _, evidence := range assessment.Evidence {
			if err := writef(w, "    %s\n", evidence); err != nil {
				return err
			}
		}
	}

	return nil
}

func collectionError(snapshot host.Snapshot, subsystem string) error {
	for _, collectionErr := range snapshot.Errors {
		if collectionErr.Subsystem == subsystem {
			return collectionErr.Err
		}
	}
	return nil
}

func printSectionError(
	w io.Writer,
	snapshot host.Snapshot,
	subsystem string,
) error {
	err := collectionError(snapshot, subsystem)
	if err != nil {
		return writef(w, "  unavailable: %v\n", err)
	}
	return writeln(w, "  unavailable")
}

func formatIP(ip net.IP) string {
	if ip == nil {
		return "-"
	}
	return ip.String()
}

func formatRoutePrefix(ip net.IP, prefixLen uint8) string {
	if ip == nil {
		return "default"
	}
	return fmt.Sprintf("%s/%d", ip.String(), prefixLen)
}

func emptyAsDash(value string) string {
	if strings.TrimSpace(value) == "" {
		return "-"
	}
	return value
}
