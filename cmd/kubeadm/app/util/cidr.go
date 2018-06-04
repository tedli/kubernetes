/*
 * Licensed Materials - Property of tenxcloud.com
 * (C) Copyright 2018 TenxCloud. All Rights Reserved.
 * 2018-01-22  @author weiwei@tenxcloud.com
 */
package util

import (
	"fmt"
	"net"
	"regexp"
)

var RegRule = regexp.MustCompile("lo|cali.*|docker.*|veth.*|tunl.*")

func IsLocalNetwork(name string) bool {
	return !RegRule.MatchString(name)
}


func getHostCIDR() map[string]string {
	ret := map[string]string{}
	interfaces, _ := net.Interfaces()
	for _, inter := range interfaces {
		if !IsLocalNetwork(inter.Name) {
			continue
		}
		addrs, _ := inter.Addrs()
		for _, addr := range addrs {
			ip, ipnet, err := net.ParseCIDR(addr.String())
			if err != nil {
				continue
			}
			if ip.To4() != nil {
				ipnet.Mask = net.CIDRMask(16, 32)
				ipnet.IP = ipnet.IP.Mask(ipnet.Mask)
				ret[ipnet.String()] = ""
			}
		}
	}
	return ret
}

// kubeadm init --pod-network-cidr
// Specify range of IP addresses for the pod network;
// if set, the control plane will automatically allocate CIDRs for every node
// GetAvailiablePodCIDR for --pod-network-cidr
func GetAvailiablePodCIDR(prefix, from, to int) string {
	ret := "172.31.0.0/16"
	hostCIDR := getHostCIDR()
	for i := to; i >= from && i <= to; i-- {
		cidr := fmt.Sprintf("%d.%d.0.0/16", prefix, i)
		if _, ok := hostCIDR[cidr]; !ok {
			return cidr
		}
	}
	return ret
}