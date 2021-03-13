package liqonet_test

import (
	"github.com/liqotech/liqo/pkg/liqonet"
	. "github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
)

var _ = Describe("IpamNew", func() {
	Describe("After reserving a network", func() {
		var ipam *liqonet.IPAM
		reserved := []string{
			"10.244.0.0/24",
		}
		ipam = liqonet.NewIPAM()
		err := ipam.Init(reserved, liqonet.Pools)
		gomega.Expect(err).To(gomega.BeNil())
		Context("That belongs to a pool", func() {
			err := ipam.AcquireReservedSubnet("10.0.2.0/24")
			gomega.Expect(err).To(gomega.BeNil())
			It("Should not be possible to acquire the same network for a cluster", func() {
				p, err := ipam.GetSubnetPerCluster("10.0.2.0/24", "cluster1")
				gomega.Expect(err).To(gomega.BeNil())
				gomega.Expect(p).ToNot(gomega.Equal("10.0.2.0/24"))
			})
			It("Should not be possible to acquire a larger network that contains it for a cluster", func() {
				p, err := ipam.GetSubnetPerCluster("10.0.0.0/16", "cluster1")
				gomega.Expect(err).To(gomega.BeNil())
				gomega.Expect(p).ToNot(gomega.Equal("10.0.0.0/16"))
			})
			It("Should not be possible to acquire a smaller network contained by it for a cluster", func() {
				p, err := ipam.GetSubnetPerCluster("10.0.2.0/25", "cluster1")
				gomega.Expect(err).To(gomega.BeNil())
				gomega.Expect(p).ToNot(gomega.Equal("10.0.2.0/25"))
			})
		})
	})
	Describe("Allocating a new network for a cluster", func() {
		var ipam *liqonet.IPAM
		pool := []string{
			"10.0.0.0/8",
		}
		reserved := []string{
			"192.168.1.0/16",
		}
		ipam = liqonet.NewIPAM()
		err := ipam.Init(reserved, pool)
		gomega.Expect(err).To(gomega.BeNil())
		Context("When the remote cluster asks for a subnet belonging to a network in the pool", func() {
			Context("and the subnet has not already been assigned to any other cluster", func() {
				It("Should allocate the subnet itself, without mapping", func() {
					_, err := ipam.GetSubnetPerCluster("10.0.0.0/16", "cluster1")
					gomega.Expect(err).To(gomega.BeNil())
					gomega.Expect(ipam.SubnetPerCluster["cluster1"]).To(gomega.Equal("10.0.0.0/16"))
					err = ipam.FreeSubnetPerCluster("cluster1")
					gomega.Expect(err).To(gomega.BeNil())
				})
			})
			Context("and the subnet has already been assigned to another cluster", func() {
				Context("and there is an available network with the same mask length in one pool", func() {
					It("should map the requested network to another network taken by the pool", func() {
						_, err := ipam.GetSubnetPerCluster("10.0.0.0/16", "cluster1")
						gomega.Expect(err).To(gomega.BeNil())
						gomega.Expect(ipam.SubnetPerCluster["cluster1"]).To(gomega.Equal("10.0.0.0/16"))
						_, err = ipam.GetSubnetPerCluster("10.0.0.0/16", "cluster2")
						gomega.Expect(err).To(gomega.BeNil())
						gomega.Expect(ipam.SubnetPerCluster["cluster2"]).To(gomega.HavePrefix("10."))
						gomega.Expect(ipam.SubnetPerCluster["cluster2"]).To(gomega.HaveSuffix("/16"))
						err = ipam.FreeSubnetPerCluster("cluster1")
						gomega.Expect(err).To(gomega.BeNil())
						err = ipam.FreeSubnetPerCluster("cluster2")
						gomega.Expect(err).To(gomega.BeNil())
					})
				})
				Context("and there is not an available network with the same mask length in any pool", func() {
					Context("and the network has not been assigned to any cluster yet", func() {
						It("should allocate it as a new prefix", func() {
							_, err := ipam.GetSubnetPerCluster("10.0.0.0/16", "cluster1")
							gomega.Expect(err).To(gomega.BeNil())
							gomega.Expect(ipam.SubnetPerCluster["cluster1"]).To(gomega.Equal("10.0.0.0/16"))
							_, err = ipam.GetSubnetPerCluster("10.0.0.0/16", "cluster2")
							gomega.Expect(err).To(gomega.BeNil())
							err = ipam.FreeSubnetPerCluster("cluster1")
							gomega.Expect(err).To(gomega.BeNil())
							err = ipam.FreeSubnetPerCluster("cluster2")
							gomega.Expect(err).To(gomega.BeNil())
						})
					})
				})
			})
		})
	})
	Describe("Removing a subnet", func() {
		var ipam *liqonet.IPAM
		BeforeEach(func() {
			pool := []string{
				"10.0.0.0/8",
			}
			reserved := []string{
				"192.168.1.0/16",
			}
			ipam = liqonet.NewIPAM()
			err := ipam.Init(reserved, pool)
			gomega.Expect(err).To(gomega.BeNil())
		})
		It("Should successfully free the subnet", func() {
			network, err := ipam.GetSubnetPerCluster("10.0.1.0/24", "cluster1")
			gomega.Expect(err).To(gomega.BeNil())
			gomega.Expect(network).To(gomega.Equal("10.0.1.0/24"))
			err = ipam.FreeSubnetPerCluster("cluster1")
			gomega.Expect(err).To(gomega.BeNil())
			network, err = ipam.GetSubnetPerCluster("10.0.1.0/24", "cluster2")
			gomega.Expect(err).To(gomega.BeNil())
			gomega.Expect(network).To(gomega.Equal("10.0.1.0/24"))
		})
	})
})
