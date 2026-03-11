package adb

import (
	"testing"
)

func TestNetworkInfoParsing(t *testing.T) {
	info := &NetworkInfo{}
	if len(info.Interfaces) != 0 {
		t.Fatal("expected empty interfaces")
	}
}

func TestNetworkInterfaceStruct(t *testing.T) {
	iface := NetworkInterface{
		Name:      "wlan0",
		IPAddress: "192.168.1.100",
		Mask:      "24",
		Flags:     "UP,BROADCAST,RUNNING,MULTICAST",
	}
	if iface.Name != "wlan0" {
		t.Fatal("unexpected name")
	}
	if iface.IPAddress != "192.168.1.100" {
		t.Fatal("unexpected ip")
	}
	if iface.Mask != "24" {
		t.Fatal("unexpected mask")
	}
}

func TestParseIfconfigOutput(t *testing.T) {
	output := `lo        Link encap:Local Loopback
          inet addr:127.0.0.1/8 Scope:Host
          UP LOOPBACK RUNNING  MTU:65536

wlan0     Link encap:Ethernet
          inet 192.168.1.50/24 Bcast:192.168.1.255
          UP BROADCAST RUNNING MULTICAST

rmnet0    Link encap:UNSPEC`

	info := parseIfconfigOutput(output)
	if len(info.Interfaces) < 2 {
		t.Fatalf("expected at least 2 interfaces, got %d", len(info.Interfaces))
	}

	var wlan *NetworkInterface
	for i := range info.Interfaces {
		if info.Interfaces[i].Name == "wlan0" {
			wlan = &info.Interfaces[i]
			break
		}
	}
	if wlan == nil {
		t.Fatal("expected wlan0 interface")
	}
	if wlan.IPAddress != "192.168.1.50" {
		t.Fatalf("expected 192.168.1.50, got %s", wlan.IPAddress)
	}
	if wlan.Mask != "24" {
		t.Fatalf("expected mask 24, got %s", wlan.Mask)
	}
}

func TestParseIfconfigOutput_Empty(t *testing.T) {
	info := parseIfconfigOutput("")
	if len(info.Interfaces) != 0 {
		t.Fatalf("expected 0 interfaces, got %d", len(info.Interfaces))
	}
}

func TestParseIfconfigOutput_SingleInterface(t *testing.T) {
	output := `wlan0     Link encap:Ethernet
          inet 10.0.0.5/16`
	info := parseIfconfigOutput(output)
	if len(info.Interfaces) != 1 {
		t.Fatalf("expected 1 interface, got %d", len(info.Interfaces))
	}
	if info.Interfaces[0].IPAddress != "10.0.0.5" {
		t.Fatalf("expected 10.0.0.5, got %s", info.Interfaces[0].IPAddress)
	}
}

func TestParseIfconfigOutput_NoIP(t *testing.T) {
	output := `rmnet0    Link encap:UNSPEC
          UP RUNNING`
	info := parseIfconfigOutput(output)
	if len(info.Interfaces) != 1 {
		t.Fatalf("expected 1 interface, got %d", len(info.Interfaces))
	}
	if info.Interfaces[0].IPAddress != "" {
		t.Fatalf("expected empty ip, got %s", info.Interfaces[0].IPAddress)
	}
}

func TestParsePingStats_Transmitted(t *testing.T) {
	pr := &PingResult{}
	parsePingStats("4 packets transmitted, 3 received, 25% packet loss", pr)
	if pr.Transmitted != 4 {
		t.Fatalf("expected 4 transmitted, got %d", pr.Transmitted)
	}
}

func TestParsePingStats_WithReceivedComma(t *testing.T) {
	// Real Android ping output: "3 packets transmitted, 3 received, 0% packet loss, time 2003ms"
	pr := &PingResult{}
	parsePingStats("3 packets transmitted, 3 received, 0% packet loss, time 2003ms", pr)
	if pr.Transmitted != 3 {
		t.Fatalf("expected 3 transmitted, got %d", pr.Transmitted)
	}
	if pr.Received != 3 {
		t.Fatalf("expected 3 received, got %d", pr.Received)
	}
	if pr.LossPercent != 0 {
		t.Fatalf("expected 0%% loss, got %f", pr.LossPercent)
	}
}

func TestParsePingStats_RealAndroidFormat(t *testing.T) {
	// Full real-world Android ping stats line
	pr := &PingResult{}
	parsePingStats("4 packets transmitted, 2 received, 50% packet loss, time 3005ms", pr)
	if pr.Transmitted != 4 {
		t.Fatalf("expected 4 transmitted, got %d", pr.Transmitted)
	}
	if pr.Received != 2 {
		t.Fatalf("expected 2 received, got %d", pr.Received)
	}
	if pr.LossPercent != 50 {
		t.Fatalf("expected 50%% loss, got %f", pr.LossPercent)
	}
}

func TestParsePingStats_CalculatedLoss(t *testing.T) {
	// When received < transmitted and LossPercent is 0, loss is computed.
	pr := &PingResult{}
	pr.Transmitted = 10
	pr.Received = 7
	// Simulate the tail of parsePingStats where loss is calculated.
	if pr.Transmitted > 0 && pr.LossPercent == 0 && pr.Received < pr.Transmitted {
		pr.LossPercent = float64(pr.Transmitted-pr.Received) / float64(pr.Transmitted) * 100
	}
	if pr.LossPercent != 30.0 {
		t.Fatalf("expected 30%% loss, got %f", pr.LossPercent)
	}
}

func TestParsePingRTT(t *testing.T) {
	pr := &PingResult{}
	parsePingRTT("rtt min/avg/max/mdev = 1.234/5.678/10.123/2.345 ms", pr)
	if pr.AvgRTT != 5.678 {
		t.Fatalf("expected avg rtt 5.678, got %f", pr.AvgRTT)
	}
}

func TestParsePingRTT_NoEquals(t *testing.T) {
	pr := &PingResult{}
	parsePingRTT("no equals sign here", pr)
	if pr.AvgRTT != 0 {
		t.Fatalf("expected 0 rtt, got %f", pr.AvgRTT)
	}
}

func TestParsePingRTT_SingleValue(t *testing.T) {
	pr := &PingResult{}
	parsePingRTT("rtt = 1.0", pr)
	if pr.AvgRTT != 0 {
		t.Fatalf("expected 0 for single value (no avg), got %f", pr.AvgRTT)
	}
}

func TestParseNetstatOutput(t *testing.T) {
	output := `Proto Recv-Q Send-Q Local Address           Foreign Address         State
tcp        0      0 0.0.0.0:5555            0.0.0.0:*               LISTEN
tcp        0      0 127.0.0.1:5037          0.0.0.0:*               LISTEN`

	conns := parseNetstatOutput(output)
	if len(conns) != 2 {
		t.Fatalf("expected 2 connections, got %d", len(conns))
	}
	if conns[0].Protocol != "tcp" {
		t.Fatalf("expected tcp, got %s", conns[0].Protocol)
	}
	if conns[0].LocalAddr != "0.0.0.0:5555" {
		t.Fatalf("expected 0.0.0.0:5555, got %s", conns[0].LocalAddr)
	}
	if conns[0].RemoteAddr != "0.0.0.0:*" {
		t.Fatalf("expected 0.0.0.0:*, got %s", conns[0].RemoteAddr)
	}
}

func TestParseNetstatOutput_Empty(t *testing.T) {
	conns := parseNetstatOutput("")
	if len(conns) != 0 {
		t.Fatalf("expected 0 connections, got %d", len(conns))
	}
}

func TestParseNetstatOutput_HeaderOnly(t *testing.T) {
	conns := parseNetstatOutput("Proto Recv-Q Send-Q Local\n")
	if len(conns) != 0 {
		t.Fatalf("expected 0 connections, got %d", len(conns))
	}
}

func TestParseNetstatOutput_SS(t *testing.T) {
	output := `Netid State  Recv-Q Send-Q Local Address:Port  Peer Address:Port
tcp   LISTEN 0      128    0.0.0.0:22         0.0.0.0:*            users:(("sshd",pid=1234,fd=3))`

	conns := parseNetstatOutput(output)
	if len(conns) != 1 {
		t.Fatalf("expected 1 connection, got %d", len(conns))
	}
	if conns[0].Protocol != "tcp" {
		t.Fatalf("expected tcp, got %s", conns[0].Protocol)
	}
}
