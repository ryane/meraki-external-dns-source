# meraki-external-dns-source

A Kubernetes controller that registers [Meraki](https://meraki.cisco.com/) device clients in DNS via the [external-dns](https://github.com/kubernetes-sigs/external-dns) [CRD Source](https://github.com/kubernetes-sigs/external-dns/blob/master/docs/contributing/crd-source.md).

## Background

I was lucky enough to have been gifted a couple of [Meraki](https://meraki.cisco.com/) devices: a [Z1](https://documentation.meraki.com/MX/Installation_Guides/MX_End_of_Sale/Z1_(EOS)_Installation_Guide) and an [MX69](https://documentation.meraki.com/MX/Installation_Guides/MX_End_of_Sale/MX60_(EOS)_Installation_Guide). I primarily use them as a simple way to run a Site-to-site VPN between my home and office space and it is helpful to have all of my client devices registered in DNS. This controller will discover all of the devices in a Meraki [Network](https://developer.cisco.com/meraki/api/#!get-network) as defined by the `MerakiSource` CRD and set the list of active clients as endpoints in a new or existing external-dns [`DNSEndpoint`](https://github.com/kubernetes-sigs/external-dns/blob/master/endpoint/endpoint.go) resource. external-dns does the actual DNS registration in your configured provider.

This is pretty niche but perhaps it will be useful to someone else.

## MerakiSource

Each `MerakiSource` represents a [Network](https://developer.cisco.com/meraki/api/#!get-network) defined in your [Organization](https://developer.cisco.com/meraki/api/#!get-network). The `domain` specifies the domain suffix for all discovered [Clients](https://developer.cisco.com/meraki/api/#!get-network).

### Examples

Consider the following `MerakiSource`:

``` yaml
apiVersion: dns.jossware.com/v1alpha1
kind: MerakiSource
metadata:
  name: office
spec:
  organization:
    id: "999999"
  network:
    id: N_111111111111111111
  domain: office.internal.example.com
  ttl: 60
```

The controller will find all the Clients in the Network and add them to the `.spec.endpoints` list in a `DNSEndpoint` resource named after the network. For example:

`kubectl get dnsendpoint office -ojson | jq .spec`

``` json
{
  "endpoints": [
    {
      "dnsName": "lab01.office.internal.example.com",
      "recordTTL": 60,
      "recordType": "A",
      "targets": [
        "192.168.128.5"
      ]
    },
    {
      "dnsName": "rt-ac66u-7d00.office.internal.example.com",
      "recordTTL": 60,
      "recordType": "A",
      "targets": [
        "192.168.128.7"
      ]
    },
    {
      "dnsName": "silicon.office.internal.example.com",
      "recordTTL": 60,
      "recordType": "A",
      "targets": [
        "192.168.128.14"
      ]
    },
    {
      "dnsName": "pi01.office.internal.example.com",
      "recordTTL": 60,
      "recordType": "A",
      "targets": [
        "192.168.128.13"
      ]
    }
  ]
}
```

At this point, assuming it is configured correctly, external-dns will see the `DNSEndpoint` and register the DNS records in your chosen provider.
