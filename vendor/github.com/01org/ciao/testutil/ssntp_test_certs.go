//
// Copyright (c) 2016 Intel Corporation
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package testutil

import (
	"crypto/x509"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path"

	"github.com/01org/ciao/ssntp"
	"github.com/01org/ciao/ssntp/certs"
)

// TestCACert is a snake oil Certificate Authority for test automation.
const TestCACert = `
-----BEGIN CERTIFICATE-----
MIIDFzCCAf+gAwIBAgIRAP/zvt9ZFbbNIxy/1o6gvIYwDQYJKoZIhvcNAQELBQAw
CzEJMAcGA1UEChMAMB4XDTE3MDUyNjAwMzgwMloXDTM3MDUyMTAwMzgwMlowCzEJ
MAcGA1UEChMAMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAw61q1QTx
A6JB+eji7FCYQrlg527GCg0ezySHSiqkyCEK17SqDS9AK9N5Ps8gGWCUR1BadD1J
PRj1KOTKGreFZysy1SPB8yuYUNWb36mrtsmijP8vT9lnVbJnYh39t0IDFxRRVN+R
VXdImAzCtsHw//sKavWVAwsMkiM6SCy/iY/TLUQrfNhaxi9uG1TxrQ3LUTiLfHuF
HChmr1EKyWhm7nUx/knTGaWxk7IVhq06aDVKVbuLABEebvwUa5UVPjEwUBKHD5OE
3gt8yLCNimQ3JlYvodZROINWr/HcRK9DaEVhkaJdI7K1HfoC/OrUplxGWf/wam6P
mRuT0xkvYs/bzwIDAQABo3YwdDAOBgNVHQ8BAf8EBAMCAqQwGgYDVR0lBBMwEQYE
VR0lAAYJKwYBBAGCVwgCMA8GA1UdEwEB/wQFMAMBAf8wNQYDVR0RBC4wLIIJbG9j
YWxob3N0gR9jaWFvLWRldmVsQGxpc3RzLmNsZWFybGludXgub3JnMA0GCSqGSIb3
DQEBCwUAA4IBAQBTG93Ll5v3uQzcULp2rYnKFHsnhGPhQkyAYRh04b5Fh74pCSTQ
xtX4vpciXNi48WmmvVVWs40NMsF2pj4s260044/71PJ98kj8M8+4EvCNEohpOxFr
13gFp3iO8UoqoEojJXygSJPzFZZ4xxb+w6LGf8MEE13N4CgVvFIgzG2j3e4OaCOz
kFb3mlHvZB4RYOgNeuPBS0hPAJZ6W/ehbME6KfiWvMLedpOLvd3a+pPEHYjpOAHc
R0CO0aRlDP2QCH/N2yXkoULCoGVnQ170Uu1UKQGUevlDNZOcOYKXTHSrwZXXTwGC
2rQwSgX4LaAqWlcyNB7zSN/eXQ7wd1qN5zrx
-----END CERTIFICATE-----`

// TestCertScheduler is a snake oil SSNTP Scheduler role certificate
// for test automation.
const TestCertScheduler = `
-----BEGIN CERTIFICATE-----
MIIDFzCCAf+gAwIBAgIRAP/zvt9ZFbbNIxy/1o6gvIYwDQYJKoZIhvcNAQELBQAw
CzEJMAcGA1UEChMAMB4XDTE3MDUyNjAwMzgwMloXDTM3MDUyMTAwMzgwMlowCzEJ
MAcGA1UEChMAMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAw61q1QTx
A6JB+eji7FCYQrlg527GCg0ezySHSiqkyCEK17SqDS9AK9N5Ps8gGWCUR1BadD1J
PRj1KOTKGreFZysy1SPB8yuYUNWb36mrtsmijP8vT9lnVbJnYh39t0IDFxRRVN+R
VXdImAzCtsHw//sKavWVAwsMkiM6SCy/iY/TLUQrfNhaxi9uG1TxrQ3LUTiLfHuF
HChmr1EKyWhm7nUx/knTGaWxk7IVhq06aDVKVbuLABEebvwUa5UVPjEwUBKHD5OE
3gt8yLCNimQ3JlYvodZROINWr/HcRK9DaEVhkaJdI7K1HfoC/OrUplxGWf/wam6P
mRuT0xkvYs/bzwIDAQABo3YwdDAOBgNVHQ8BAf8EBAMCAqQwGgYDVR0lBBMwEQYE
VR0lAAYJKwYBBAGCVwgCMA8GA1UdEwEB/wQFMAMBAf8wNQYDVR0RBC4wLIIJbG9j
YWxob3N0gR9jaWFvLWRldmVsQGxpc3RzLmNsZWFybGludXgub3JnMA0GCSqGSIb3
DQEBCwUAA4IBAQBTG93Ll5v3uQzcULp2rYnKFHsnhGPhQkyAYRh04b5Fh74pCSTQ
xtX4vpciXNi48WmmvVVWs40NMsF2pj4s260044/71PJ98kj8M8+4EvCNEohpOxFr
13gFp3iO8UoqoEojJXygSJPzFZZ4xxb+w6LGf8MEE13N4CgVvFIgzG2j3e4OaCOz
kFb3mlHvZB4RYOgNeuPBS0hPAJZ6W/ehbME6KfiWvMLedpOLvd3a+pPEHYjpOAHc
R0CO0aRlDP2QCH/N2yXkoULCoGVnQ170Uu1UKQGUevlDNZOcOYKXTHSrwZXXTwGC
2rQwSgX4LaAqWlcyNB7zSN/eXQ7wd1qN5zrx
-----END CERTIFICATE-----
-----BEGIN RSA PRIVATE KEY-----
MIIEowIBAAKCAQEAw61q1QTxA6JB+eji7FCYQrlg527GCg0ezySHSiqkyCEK17Sq
DS9AK9N5Ps8gGWCUR1BadD1JPRj1KOTKGreFZysy1SPB8yuYUNWb36mrtsmijP8v
T9lnVbJnYh39t0IDFxRRVN+RVXdImAzCtsHw//sKavWVAwsMkiM6SCy/iY/TLUQr
fNhaxi9uG1TxrQ3LUTiLfHuFHChmr1EKyWhm7nUx/knTGaWxk7IVhq06aDVKVbuL
ABEebvwUa5UVPjEwUBKHD5OE3gt8yLCNimQ3JlYvodZROINWr/HcRK9DaEVhkaJd
I7K1HfoC/OrUplxGWf/wam6PmRuT0xkvYs/bzwIDAQABAoIBAAh1aaXVtdlzXSjB
cXXHsh1ISDEY78Sldox7xsFlAISKMR7L94HkZgC+/oHBkGCodSB0D8TwlUbn2kkv
QrFO95xTGLpv9kVdwBLWeQt9GSgopTc1HMV132qr8J4kL8CJQPrxbOafV3f7VQ8F
ljEyRwm5v2SKQyvDgYKbtTxDevAmTQiFZ/4AQE3cn0/dfNyl0Fd9gNgS1grh8pfc
mxizFfAStCoisjsCZ+LGPRCiRNvL5D0Bwb5l3dPGLb0GziVjc0/oBsICUxyE5r4f
HsQ1qwvgeVpGFHwyFwOpIbRL0xL4TCALh2Ysu/MB2go67j/cQ0qPRSoKZ/eE6Pre
PD/pNSkCgYEAyw9OPTpVdwkZI3ZqzZfEVjTAZDSm3zhE9iOvbuBXmR3P0eZq44DC
+DaXu7FeGu7OE22f4DJZ+uEtBB5MMujREHWlij949jMCkP0TGXrt7a6XtJFDGKJg
cTg/5TaGI/59sDw14WjuzFJe/OAukGL7iJ2OVdMwnMcNLySvX3qnb3UCgYEA9rFk
8O8VbobgJemb6SdjktA3bEQnl134c4tuvPS0wZoCKENZKi3tk8GTc96WGq3WLYSe
Q92QcWOy6g8/BNW6bLhrHslRqw5vAhz0ZKs1p2WIsZOnTbTl+OsThlVRwLsS3dob
1sCgecMJ/G+Z+btsvu1HnBlQYcszIEvOpbGgmbMCgYBWEtLTWVrI7m5dfeCf7Wko
MYwr7bWegTeaLl463ZXELcLd8pH0hawfkuSWhwSg3gE0cw+F9VH26mQujrk2C0Iz
e+sDwwv/MHgyBVSHRHh+e7eKrtiGJK5Ez9clzgrmTwXwIlWkitpOecwR3OVgBtUg
f8jJ0I+WpTmNdjtweYln0QKBgBg72PSqJ+rRqRdQWZaP3gJAHhGuqE0AWDXRjrFV
QKR8IpYd95ZjKKGJNJj/VrOMPCwAiSOVkmjxKFRB5yjsbgHcI/nEQReStWj5uzBg
eUbWfJUlMhw6FxVa0nIx03QhbHsKwA1aoukTNdnshK25sbcXzB8ThYf11DHqAITa
bDJtAoGBAL9okagG9PTXEaLuEPkPr7JAaRwT4Kaf+UnB+gMIyrTc5fClqK+6ki8b
DbHPI0yLGuYBNZ5YWwmO9OW2BEvzmZl/f+ZM7UMBX/thJMII4ENnF4ocZ6OfndU2
ALAJflKpxgLudxzweoFEoWdZ6uVhhQMbITH9Hh0YT35mPSEglDFa
-----END RSA PRIVATE KEY-----`

// TestCertServer is a snake oil SSNTP Server role certificate
// for test automation.
const TestCertServer = `
-----BEGIN CERTIFICATE-----
MIIDFDCCAfygAwIBAgIRALUnnqHjCZ0GRKZh8r9pjPMwDQYJKoZIhvcNAQELBQAw
CzEJMAcGA1UEChMAMB4XDTE3MDUyNjAwNDMwNFoXDTM3MDUyMTAwNDMwNFowCzEJ
MAcGA1UEChMAMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA60CUJfon
VE7hWYrY7yZyBdQ5UFq43kelIdN7plKv84fIagRxlOvi6NEFjnwBrAEhaOdhXBZl
jEBWg8kLem/+iNBBi0TL+W4qTPDLehwHmaMiOGvJgsuQksPn1tCB/tZSY6ONdlV4
18tMllHVSVGoRpHbLvMTu4lsyiRdUb3a4U6pQfZb9XWJILA0CZih49TKh4rCkRnS
mLjWsJngDWDwKQoxhywxCQLIp1loYrqpbSQQCbGkh9LwCz+ASiusfU794t/6Sm/4
8cZxjusoXnEjmiLDCEJ2WrYJswzeyTUVqvlCWmnzOTTR7X6PxXkdg4LT7EgoJIkh
lO107pFBnlLEBQIDAQABo3MwcTAOBgNVHQ8BAf8EBAMCAqQwGgYDVR0lBBMwEQYE
VR0lAAYJKwYBBAGCVwgFMAwGA1UdEwEB/wQCMAAwNQYDVR0RBC4wLIIJbG9jYWxo
b3N0gR9jaWFvLWRldmVsQGxpc3RzLmNsZWFybGludXgub3JnMA0GCSqGSIb3DQEB
CwUAA4IBAQAMqKK2eh0RJLraICE2Na3t1prsj0xhFv9LwpLip5lw6iDXzeYTalkR
12Ls1MmJ3yvV3x/1QS4u/fhN/HB6P1hVD3Iv7QdTZba8kgL8pXAZWku97VfE9yka
nNnfm2eJ32fMvQda2EqNppdGX91VDsHZfh37fNnXooqbB7cfR6RJpHqzYyAZc3Zq
wwEeJ3d3LXXhuasp6Rk9b8NVRuq94lIp8MykIWrpISUk/oM5RqX8Ah/rAUZDk3ZI
QQJBxpFiLELiJbyxs7ecC3ST0j3ISwtgeDVebchV4uF5JquTIc+9wv7u+/Vva2/7
zaKG3aaT5UHdd1YqelMWi4HhXOTo8gwI
-----END CERTIFICATE-----
-----BEGIN RSA PRIVATE KEY-----
MIIEpAIBAAKCAQEA60CUJfonVE7hWYrY7yZyBdQ5UFq43kelIdN7plKv84fIagRx
lOvi6NEFjnwBrAEhaOdhXBZljEBWg8kLem/+iNBBi0TL+W4qTPDLehwHmaMiOGvJ
gsuQksPn1tCB/tZSY6ONdlV418tMllHVSVGoRpHbLvMTu4lsyiRdUb3a4U6pQfZb
9XWJILA0CZih49TKh4rCkRnSmLjWsJngDWDwKQoxhywxCQLIp1loYrqpbSQQCbGk
h9LwCz+ASiusfU794t/6Sm/48cZxjusoXnEjmiLDCEJ2WrYJswzeyTUVqvlCWmnz
OTTR7X6PxXkdg4LT7EgoJIkhlO107pFBnlLEBQIDAQABAoIBAQC9OtjwI2P3YOvL
hvAwjhAxuB/SDuedhKvDpcVUaDa4AYSoIqLqU0PWWivKDN2bad1h8JxT4oAUbLwq
jVD6T5PCoSHX0KLyJDdKZHaH5nwGjT49fBY/a1cDdynJlTa7sdHb6/ciNGZbzl/w
miqiK1jcSv6vqT86HrSvdMjLs5eYmo5s9ygf1sy1AVa6h6gKT3rcBcftc8TtLZM3
nGO+R8sM1UdwzruUss/bJgPnWfFtVW9EE8byX9c6uLJVRjgAhYv8sD15l8BipVFP
Fk0yruMQatrAxFaO9tis6SFBSQJyITMSRVKbDOYscAGQLGNXVE1jbidUr/nOvf3N
745shSGRAoGBAP3KBczVJgLKA+k2RlRKC5ETHzFMk4Ulj4IXsoXYHkwXdsGckY0e
vk8MMHQdUK5pPrM5vXz55f+wVMESKFX2AVi6ozFMnX+IlyPL4olNlzFnuMDg7oyV
t8tj7tGkak3LIzKBdjo+YSFWQHGqixL/2ya8uD3aLWGSrvPdmS8CSjXzAoGBAO1N
N35KLSjZBfBxO9/XP6Hxg1f8z0AtlsSiTqgloCVHmEXLESY9OCgv4wx8vxO1LC6/
qTLjV6utBrG1fq33QrTca9JTcN/Ol/cMIZmuLDiigWtcQ9AwvGUaBUE1wihNr3V7
Ftucyy0AG1ZgyqHG4tdRz3W6RK49w0osCmhykEQnAoGAOz2nMPMoVkpVs2CJ9i76
mDjAdT+Mx+3Gm/VwJLIYEGcBv5wOlcRxY/5SaShWpv/GNQvrYXrr501/2zmj1L0B
/3ZBlcZulVCLBz4WeTp1aoDtrYhT5tkj+AQxwRoB/nrGkomJ0XqyLZf2nxHSOPMk
ctxmnXmKUlZtJFu74C9Gp2UCgYBWW6V4VjI9DU22BN9PRJwpqSStXpllt7GIebC6
TIcNShLGQ3JIQjsvlM3B+5vl5ibgFGvU0xtSpLMs9OnXEYa7HwQ2FJudNyfihg2s
SdBaA/mpQniDSVkmSePjqVaxKCRUUqks3tCp3cIVG0Biw2hGB8XCCDl6V4u8cG6R
OC/8PQKBgQCzMQF+hOEOk0hceR3jGWf25jSFQiE6OikeIL7mcyPdKR6hS7LEFYRW
f91ni50xiWfog4qoq78aNA9DtQuCS3rdMaZOKokpUbuIIStTCVIxmPup6O1bZWoF
dE3jdW+ROHJQC7WHYcwv9ByIJZ8PSCyzok5KZRYSblbBQocpy6E/xw==
-----END RSA PRIVATE KEY-----`

// TestCertAgent is a snake oil SSNTP Agent role certificate
// for test automation.
const TestCertAgent = `
-----BEGIN CERTIFICATE-----
MIIDEzCCAfugAwIBAgIQdznBZctv+KsfIGiu4RXpXjANBgkqhkiG9w0BAQsFADAL
MQkwBwYDVQQKEwAwHhcNMTcwNTI2MDA0NTQyWhcNMzcwNTIxMDA0NTQyWjALMQkw
BwYDVQQKEwAwggEiMA0GCSqGSIb3DQEBAQUAA4IBDwAwggEKAoIBAQCutP4sSVA9
kaBUGKMbJOXB6kTlybzovMpvkB5qGjpPTGJUQ9TtN9voZcS0xsx4nuSXqyOOm+v/
ByKcOk/IkfA3zBU5rr5s5AC4qChlLxuv8DT7104Xv8LqbYd/bRaQXvKrxBdUIoH/
F1G1/a/pAmvZcHFZsoyfpywidN9xgcib5xR1+Z3SQWWJL0KjDgRPR6pg+4YuELyk
+U/CwMpH6NN4fWwsgc/mOTV721OT7GXbOLHzpvQVcJEa0f1HrMPkRrscMTBjQznY
jf/FF4wotLY7rVLsLsCc77fKd285pnmCMYn2JgUohNr637bbY9Nx99/OPmvmPW96
pxs7qS1VDn97AgMBAAGjczBxMA4GA1UdDwEB/wQEAwICpDAaBgNVHSUEEzARBgRV
HSUABgkrBgEEAYJXCAEwDAYDVR0TAQH/BAIwADA1BgNVHREELjAsgglsb2NhbGhv
c3SBH2NpYW8tZGV2ZWxAbGlzdHMuY2xlYXJsaW51eC5vcmcwDQYJKoZIhvcNAQEL
BQADggEBAKQuWe6fnn6QV6uxWwyNPOI+f81IFJ1RrKPhhiAdu2n4BZUt+eyTx4R5
obkcj/053P55m0sZUtGss+H1i+TgRPZO4Dpde0gKAHMkxz3Q8xDv/UNQlWtyObTx
sqW1OHbzP5tpgAxd+8rUYe0k/3gWZTVM9ItYsDjBTyQIjxVXhjl3L0GFIZt/yEuW
l5srm0E8SMkrRZ//HEtXFPoXfutkoGtBNx/cHCrpPIHwWpuEmB3TAdizHHACVoBN
UwCbMUKypNiCYJh6EJBQ9klPu0zT3KW6LdgIlIALtymHLY8t9lhETDspGgR3GnZD
MdVyO9IOZ+hUo8SiLxIqJTPtC9j2s/I=
-----END CERTIFICATE-----
-----BEGIN RSA PRIVATE KEY-----
MIIEpQIBAAKCAQEArrT+LElQPZGgVBijGyTlwepE5cm86LzKb5Aeaho6T0xiVEPU
7Tfb6GXEtMbMeJ7kl6sjjpvr/wcinDpPyJHwN8wVOa6+bOQAuKgoZS8br/A0+9dO
F7/C6m2Hf20WkF7yq8QXVCKB/xdRtf2v6QJr2XBxWbKMn6csInTfcYHIm+cUdfmd
0kFliS9Cow4ET0eqYPuGLhC8pPlPwsDKR+jTeH1sLIHP5jk1e9tTk+xl2zix86b0
FXCRGtH9R6zD5Ea7HDEwY0M52I3/xReMKLS2O61S7C7AnO+3yndvOaZ5gjGJ9iYF
KITa+t+222PTcfffzj5r5j1veqcbO6ktVQ5/ewIDAQABAoIBADeqr/pIeerERgPF
veLeRN8e2EknmKvHy/D0SNyh8sZlnkcfPe9ABy/rjVvUpD4i0s+I1lGQWQfvrBV/
dwB/j70XqAOzLDXiCGDOI+Dpu7a5oQhFuDpU/bRYpf3yMmhZ+JTGbHCAdk9jjMOi
S7TA8sBb1aIxBCGy0JtCBhhStCsIVWK4I2ArR/ZAspeW4kOJ8F7kSoYeptw7K7U0
50DJQqAeUltdOQWp1lgfnQsorR0e5j5UHFLUVCCnvrHMxLQpT4C6f80umHLcj9Qb
w8yE83USJNSg238tnvvxJFylLtWL02UTtRt63GFyNemT60B30zba2RvIUHK0yoY3
3Xc5LbECgYEAzKtlIe5e5EknlSHiJ/SzLNc97MZVIazHUEa5Jg7Gc7IMETPbg1Ji
gnQJETMyVZS0qUjLQTy1gqQwduxkh3DaO6TtqoLIj5UpyNe43AdqFblwLS07SneE
9wgfQ/7e24OCSPFYeQ3m230bZNtnJpD5nYTsGUlQAXOWNeAYmrDtJk8CgYEA2oXi
Zz3cRoiSo7fUelKiLEXXFNPSTDRdJtUZUFkyzxLCCmgXpPssr7iTy7jmSsFOLIw6
30/iOcajGhyq/ctMewcKog2D438hlraytv3m3E9E8hNS4wiLdX31y8fj/eDvbrXp
pLsRbxJ9flte0Ed34C28vALiJfqk8w0f432UNRUCgYEAwbhsxdwIZw0y8P4cQHNl
cEjerRDgnTobgUkfj/0mK3XX5CRwXnEJGq7XsjcCKmzRPvXOpJXgu6HK2ZVQZb4U
YaXu6phVW0n1PcuphmFiMOPPYINSfl54NRWz+jjwGVf1ZjNB6XqWCyP0XNcqYB+S
lFyu2BRDLMyJ3b6ZqzlRjhMCgYEAnXYMBkjVCR4wTDiSqvIQWcaZjTB1QOQam3jC
nNspeX0SxVzsbL1xHc3q8clyaDuSkRca9P8jDG7N6Grv66EqoxwX1V3Xw35APdG4
RZP/XpDgJW83MtFdbHQvQX/wEWicHzKGAWWq0laIhxxf3cUh5DAQ54lMXAGYCmtS
pyI+QWECgYEApoJif85DVG9d/fby72RwT6FdPHKxr/xdzAUTucdWya7RKfvKxKU2
2y8/k1m2IQjBlfQSqoBoBGVjwTCOsJJegkjZHvZCkxLenHYwKCdKN/5yzfwP1mca
+YQDLLXWR4ChtTflXbClfAonUVG2RUZMUUIq6ySQLVMnX57c0BdVxkQ=
-----END RSA PRIVATE KEY-----`

// TestCertController is a snake oil SSNTP Controller role
// certificate for test automation.
const TestCertController = `
-----BEGIN CERTIFICATE-----
MIIDFDCCAfygAwIBAgIRALmE+vZGP4oYAoDkS93DxkIwDQYJKoZIhvcNAQELBQAw
CzEJMAcGA1UEChMAMB4XDTE3MDUyNjAwNDY1NloXDTM3MDUyMTAwNDY1NlowCzEJ
MAcGA1UEChMAMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAypbUIoeR
47T55fame1AJyjPi2k+VDji7q+e3FR8CJ9jGffrEJF/09PmBFC5SeRcM1cif1vdQ
aHt0VjgcgOThSRfMZZX6dA7qzDEGIlLZIsRyQo7ztPTkpYVejoB7wlkuXK+VEy/d
74bs81pofKMrxlZ72OaD8nY6UXnydLTB4Vgsl6b8csXkQN74bQvD7lQUidjbR0w0
P1Hopv+gbzf1lnv7gO6IbcRibWZ7yJmuRFU0QyomEAp/yrsHvb9Q/2XqXH+2ouV/
KETAgKaLNsfELM9JW4lK0WtJBYwtBQgZACM26UWSFv9RcYtPQ7pjG50fM+8qLApd
3++1DUCLsThl8wIDAQABo3MwcTAOBgNVHQ8BAf8EBAMCAqQwGgYDVR0lBBMwEQYE
VR0lAAYJKwYBBAGCVwgDMAwGA1UdEwEB/wQCMAAwNQYDVR0RBC4wLIIJbG9jYWxo
b3N0gR9jaWFvLWRldmVsQGxpc3RzLmNsZWFybGludXgub3JnMA0GCSqGSIb3DQEB
CwUAA4IBAQALqVFZaBpSfm07bVvlY9ZlqnXDn5x3hR+tRjfkr52Iojv35H1wD7sq
jgHyZZumKh+q3fVCq/0CGGdo4TJe71LuLBcSu5LIlDjX0BtKJZ+fnlLIfgWvHc0H
DhJeaCjNupOTstvYA6RkvOdhaWSfcdZjEAbFaydotyOSy/QvNrBwzF8LJBFZ2dyT
bffEHVlbOw2DgoEN25b7wvmkdy0MkmmRvus8qnQo8fA6rsXe6PnffsfTRs4pkQUF
lKjEC/U/cmrbIg5A30ZgPJRp20gBwSs7DilNE9lQXfJ/mnPDnaMNQJMDaQYrs62M
nGRw9BMIUxhDZvk6usinIs1QxEBSwckB
-----END CERTIFICATE-----
-----BEGIN RSA PRIVATE KEY-----
MIIEpQIBAAKCAQEAypbUIoeR47T55fame1AJyjPi2k+VDji7q+e3FR8CJ9jGffrE
JF/09PmBFC5SeRcM1cif1vdQaHt0VjgcgOThSRfMZZX6dA7qzDEGIlLZIsRyQo7z
tPTkpYVejoB7wlkuXK+VEy/d74bs81pofKMrxlZ72OaD8nY6UXnydLTB4Vgsl6b8
csXkQN74bQvD7lQUidjbR0w0P1Hopv+gbzf1lnv7gO6IbcRibWZ7yJmuRFU0Qyom
EAp/yrsHvb9Q/2XqXH+2ouV/KETAgKaLNsfELM9JW4lK0WtJBYwtBQgZACM26UWS
Fv9RcYtPQ7pjG50fM+8qLApd3++1DUCLsThl8wIDAQABAoIBAB8eihl/v649t4FW
oP4iLk7MJ5WnUdssZc+jOWFaMQeT6fGiGo0H3GXhCa3i67JEEymntr3boZNbG2S/
G8nE3sJOkIwuPJmlTPXuteWB2m7XxEFrGg5668BtOzgijmAtOMzt/7VBzhKkJDPB
eHlkyy2dTUrlJfGRraWkWNUKixmlHpNCXQDnsulDlrXW/kv4ZV7oiIo9S2DQQp0f
I7be5VKAu0WSkIn1aJpB+ormRFWuKP+8oCyzaYh5ZkPMMzLUbc8aNsE/D7VUxHtl
HyBORwC6cbHXId6HMZmlvM6PEy8pmv6U11NsofjIJFjY/RLnT0AaDKGSA/bmIz55
cv3sRrECgYEA9XGirQrFaSPdnc1bY5IZtzyTPxZMQkltpxhtfkdALc74pXzKyb9c
9g1WoXPQK6yrXW4cV/uvtzwhJIZr+qz+mCKYdYlnXMjc54RtTm+A3iTtdtLRiL1y
bcAT6+ZQ0+F0C1CXRXmxn1hROBslO1AmIDw7G/931xQ1/pT/fYVKzekCgYEA001b
p3zWGCrnD3fmLphtfqhjgQNOlVFR/9zMcp18804CSVGtv6JgiwbDhKlye+fhnVAk
kRdKHv7F3Kz6AMagXnhBlX0hcceHIXsgojtT1Zplb3P4vPmtH7TeSrmPU5vUKpmT
OBYE2u2PtW7JCKvyoYDWcWkAjALULm5eq0QuX3sCgYEA38dDclGnuzygCgf5ksbZ
+16XQaWq0aTw/LAg5ElCEoHp4bftjBOVRiDTI1DcM3Wyp/SEkxM+GeoQraSBPoQL
e9nO9xrXypi4D72Fi0XOULuKZhPARtOzSK0ffKz4dLXRf59yzD0v3QBAzM6zG2jv
2eQQYG6DbO1YbUybxG2KzkECgYEAi644p3hbovBBfDU7YZP71d2EoZVJDmYKecRB
FodLQR9RXZxz6hlyDpVzDDBjcMsxlqeS9KLbqa+rppxmS7sB6lE+sY5dXHSUvKpD
QVtMqQh+g3W7eVjne+05gVY3DAMX9u08p7fOj9a4yCwrEuNv6hlcKO5LoUKBdwwY
4siYix8CgYEA0xJtePHWno+7EZD77LFjMmKR6a0Bg+hUDZqdT241Frjyyp/1nqFw
awAJzYkv4tXyMmjkkRZcY+iyE/hojPLblUHnvHIcifFjJ3iOHoqWNKkWtIyE38Kt
zS2WwhVjI4ex3bHzfQ0bJBxWiDA4PnluD2Tm4qo1kfaoYn+Z5mFTl2U=
-----END RSA PRIVATE KEY-----`

// TestCertNetAgent is a snake oil SSNTP NetAgent role certificate
// for test automation.
const TestCertNetAgent = `
-----BEGIN CERTIFICATE-----
MIIDFDCCAfygAwIBAgIRAKKUSmhNEbjEhyh9d4FID9EwDQYJKoZIhvcNAQELBQAw
CzEJMAcGA1UEChMAMB4XDTE3MDUyNjAwNDgyN1oXDTM3MDUyMTAwNDgyN1owCzEJ
MAcGA1UEChMAMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA67eNWaLM
yhJCvw7C0U0Car1m1QlVd3ykONnG9AvT42U/fLCi3nPG0O4kq3BFOeGsPYiRzDuY
c7l+g0IF1m44F6yqVkgP3ozfoNEfExYdMQdvGsUo+BXMNgQOaYmKk7ixcAGxRlwI
u+ILe9c6DKPFjpx0izCBfuJcHSrPCRRTFjxSJSlurnTX/vYzcopbV4+RTMTeZ1qb
GJFxsENyeJdmc+pjF9ul6Mfvite+p/jFrO6VMGhIRvfQRhppcsUHE7bu07u6+yyD
Iya9jtUrk0fEeXV7IBMCfsrn/m/2IIgUNovX0w5UlbFAuYWfSPYZsTW59k7gn4oF
TiEouyc966n7jQIDAQABo3MwcTAOBgNVHQ8BAf8EBAMCAqQwGgYDVR0lBBMwEQYE
VR0lAAYJKwYBBAGCVwgEMAwGA1UdEwEB/wQCMAAwNQYDVR0RBC4wLIIJbG9jYWxo
b3N0gR9jaWFvLWRldmVsQGxpc3RzLmNsZWFybGludXgub3JnMA0GCSqGSIb3DQEB
CwUAA4IBAQAE2qZHOaOAsw5W/fLWW2vH+ug4KleCy3sIP6RrsiMLCsovZ9q1UH+U
qABHf8AiiEbZmSQfPWmXzWLKwkc/ZPdDGzi9sdlfHC38MdT55mKSQiXsk6KwbaMx
2tXssJtFvvPNZoyAuaIUz6iEixugZynuiOECehk+KWBDTdu8QjrJkIKWk4+vl1SM
OB5SPat+0PxYAAZbefRcIHtB3tXk2V78heVjukdjfg/DGJnCOY+58ox812y4muhc
URTchW88/0LRJwQ7y4xnUyPihENEyUrr4l0GNzD4h2uk4FhUwacyddtVip/DTleC
pnP6tqbhiScx74ysTD7SrcNE5shqhMjS
-----END CERTIFICATE-----
-----BEGIN RSA PRIVATE KEY-----
MIIEogIBAAKCAQEA67eNWaLMyhJCvw7C0U0Car1m1QlVd3ykONnG9AvT42U/fLCi
3nPG0O4kq3BFOeGsPYiRzDuYc7l+g0IF1m44F6yqVkgP3ozfoNEfExYdMQdvGsUo
+BXMNgQOaYmKk7ixcAGxRlwIu+ILe9c6DKPFjpx0izCBfuJcHSrPCRRTFjxSJSlu
rnTX/vYzcopbV4+RTMTeZ1qbGJFxsENyeJdmc+pjF9ul6Mfvite+p/jFrO6VMGhI
RvfQRhppcsUHE7bu07u6+yyDIya9jtUrk0fEeXV7IBMCfsrn/m/2IIgUNovX0w5U
lbFAuYWfSPYZsTW59k7gn4oFTiEouyc966n7jQIDAQABAoIBAGb86ZtSUBux4svL
TT9ZYEb2vekyjM8J/E6CiDS0vj1KTXTTUDXVa/Z5NjhZc0WY3kJ8Wwdaun9Feosq
25YWzhc576qHDbf04PhIpkUWkmaLkvWlUwMhsvmeyBVAbPWh6pS/iI7vQzmjx9Sx
8sD3BSgMH7d41/tyN0DfJVoYMT0zAxsphkasqnA2YvmS2PK+tsnH6Jvr0Bdy4Cy/
pYePq9S5bE4/GLdBSqCyBQrShMgtXn+eXnv30rhT18Bod6zlfnRCXAGoc/FOnlEC
YzqLT5MNH47uwzwA7xb2STrZafAbyEtNTl5AT18XJyO7l0ZAxWh12/lC+sPIWYe/
JVrATZkCgYEA+RzlVenCAb7QcYIvTIXIf6xpsrxq28AnzhSJ3z3vRSmuFX3TqsIN
0Lx9LmsW1kKG2iyzXnwo5Uj0+YjMY1VastGF53Vcg5cUxk/9ogW/W4sPy1jUk8Ja
VQ9oKhLV3cfldaUqSzf6fT2D5zBKprklgnO820vyPmIIHa6vJg2fydcCgYEA8jvY
ujHe3hMUlNNtE72FiiyyVhCRQ9TD56qIJmVBjZc4ILAejlNJOMToKgX+B5ECMnMd
TPlnKslN/X4BOMAVkzvME5uEBq8KQzTk1qojKuGo6cPu3H/t8dQJNTyAnzxxardz
n2h9cOXtmeNAzHtcqRQAd4V9YnqP+Yrhb/MEYTsCgYB3AEnB19AY21lhz+neaU5V
Rzya6I03erzJImCWZ1TEultx4tDZkqfc3h4CrZ+ULOWUlaP979vtZAO6rJHOpfiU
0ahg4FyYc/S1o4KrAoneJjkeT8oE5+QVHC4LY0INFy/TGlpw4kXjzB4Vs6kFqg50
GevO6qHHETeFTmxXBk0dswKBgFTEaKVPymQAXVVvX15nFhIybf38MjmAfUXWwWpe
SMBZyMR3nVnE/3ykO3JpQmo3boNlET3ckSPB6k7pB1hqr6IkbNf3tg34tyipm+Mb
Cs94xHl5nV8ATa4wu0Ar+f6/Uhk8NXP1RuB5NdqCUiy8hsKMQ1WQGz6ZEUUMOrPI
YSH9AoGAK6Q+Y2pa9MxmjNP8OeNSXnlfgWISaU7mlvxfS1hFMHzDRQIC7Da/WAjM
qqhhd2QClmd60AwSMLQ6izg//0o1eg5tfn1LfoohJOtTPbhqefROLCtjYIq9JFg5
p5dqBZa/j0qCVammetWGNPDOkC1f2q4J0YCKjQxo3ZZsgnhhbsQ=
-----END RSA PRIVATE KEY-----`

// TestCertCNCIAgent is a snake oil SSNTP CNCIAgent role certificate
// for test automation.
const TestCertCNCIAgent = `
-----BEGIN CERTIFICATE-----
MIIDFDCCAfygAwIBAgIRAPBXG2mK3BFqmhMD9sh6dQ4wDQYJKoZIhvcNAQELBQAw
CzEJMAcGA1UEChMAMB4XDTE3MDUyNjAwNTAwMFoXDTM3MDUyMTAwNTAwMFowCzEJ
MAcGA1UEChMAMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA7JSAntFs
w/DCzZaUkfJfai7fl43SHPmr2kZaw9SfoYyI30xeBuJw6l+DCHB/6MqwY75ehAAI
Lu6BfircZBtedulMsv9xMHbP74k2ihKIU1xrlidPVguP1qygjaD6a6+Uw7KeENQO
hf+X+LWZInyNGQ0qHbxwnaigM68NOgQH3bM9FyYnPz0HBYbUffGpC1RygaWTndBX
tR6fZccZ5wj4Cv6updof2LZorZeUkO2/rEwkBExINGFFo0yIs1Kvs8HKIgLF77pX
oBBLCi9nNCCh/n9gf9K+JOBmE1EkB2MbrNtaCnVk9p1/FurQOI9Da6cBVmEJO7m0
/ZR5x7yFIChkywIDAQABo3MwcTAOBgNVHQ8BAf8EBAMCAqQwGgYDVR0lBBMwEQYE
VR0lAAYJKwYBBAGCVwgGMAwGA1UdEwEB/wQCMAAwNQYDVR0RBC4wLIIJbG9jYWxo
b3N0gR9jaWFvLWRldmVsQGxpc3RzLmNsZWFybGludXgub3JnMA0GCSqGSIb3DQEB
CwUAA4IBAQAo/2oDP4SzisFUogCEKDCeX7ayLOIEvgvff8XMA+YU6bhVDyia23zf
FK0ghtxKSkKj9kwLamQXd7iAGnjzPgi1qtl2QlfEJsPW5B5o/uPDHenCIXRuz2Ua
6rrOo1DQaimIVTtWaLN72ccME/d14ofd0abrYEhSiTYCVserCr7ni7KZxmXyPCVK
a7MinG8x2EuURYH9uf9pVoTXWv6wy24iNpug3sfPFJwjTEguZ2YMTXoTDphQfsb+
2R/QcYjPlRpnq3bsiIxdcnIB+moooqIdqCY04N9XjO8rd6hT2al0zB8Vsn4Zzqmj
pr5EyyrlwhAOYuREutbgoSaRwHwXhBn1
-----END CERTIFICATE-----
-----BEGIN RSA PRIVATE KEY-----
MIIEpAIBAAKCAQEA7JSAntFsw/DCzZaUkfJfai7fl43SHPmr2kZaw9SfoYyI30xe
BuJw6l+DCHB/6MqwY75ehAAILu6BfircZBtedulMsv9xMHbP74k2ihKIU1xrlidP
VguP1qygjaD6a6+Uw7KeENQOhf+X+LWZInyNGQ0qHbxwnaigM68NOgQH3bM9FyYn
Pz0HBYbUffGpC1RygaWTndBXtR6fZccZ5wj4Cv6updof2LZorZeUkO2/rEwkBExI
NGFFo0yIs1Kvs8HKIgLF77pXoBBLCi9nNCCh/n9gf9K+JOBmE1EkB2MbrNtaCnVk
9p1/FurQOI9Da6cBVmEJO7m0/ZR5x7yFIChkywIDAQABAoIBAQDopiDWDZyoE1t+
UVZJL9Ak23OF1jGJzPzy6bzYV3+jnk/7R14v5v6jfMmewwMGKkzLyamopV9mx6UQ
LZYN29xJk6OZYxosTqqtJII9xXvKflhOkNm0BCqvMZOxs1yQCVqCGGTYp7CglXkd
W3f8Mf+PYyLHm0gjwm/IY5zeMJiLqpNHLJtBRnHrm+QVFkVNRj/r7oVf2/7f+BON
Vc59kSkaC00HqDaTrRtfJwc38bGi7lnhvr2Di5xcwuSwbk3iCaDjI71JMWdNnOBl
uvpeaNhcrw+XB3EtBOA+dWQebXTrIbyUi3KSQAW/IZG3aSvHHxaQWfhJs+uI5nav
qpr8LAgBAoGBAP+4YGGjKQjLUNkPOAY3o5yva0XFMuHjpce7opfWuo24PSBbIzzN
Uf2TGIMAAI77Gb9xCmDx05kSg6wi0bOXlul1waQdvO/MgS3HibSRvB06xNYqSX2L
cXGBDnEBiP9tyu1cx7oH/3d/ta+POe0Oh5KaYKhkHbAUobD4n7F4nbexAoGBAOzW
w9sJQk/jNm2jHfLD4hoB1jMYi8wD1/hwBByNguKm8vk+TOJMVLlazfqQMSf3fSV2
8O/k9stri1Wfrn85DiOb90eFSLdmvG/fI2jPP/KQ2ueb3RzM01HbY91mIblBxUWR
LH7VgLLwSGGkhgIGY/REEiOwB5jg/v0XLdE2Yr87AoGBAKC9WcAl2kZP3tsB6Ppn
gO2dinWJ1kkNWoipFjQRYqRwmeO7xfOTMCWPj8nQd4lopy+iM57qg1Jlw+Sw4lXc
RJ0tSvIJS1kEmHKZSaL6NF+/MDlazWUgAMgTEmvQRjgg4HzBZD44hsmruh3HjubG
yktJxNY0UED9RwHB1ketBJ6RAoGADIWaW1VU/TZNJWTPa4txw+A++/qbQZEedRMv
FHdi6Srcg9MIa5qPjDFB3LKM9sj+A+ITAQwBBGZOOpuztSRGHBnd7Bke7BtxcRTC
IYN7pQ6FlGNIQIKP1a8cy5Lfy5Svomr3iEkvgcZ0fT0enLLLzBlhQCPJcwrKUIVO
NdaDSAsCgYB2D7iZ2w8NvqlpgBF8qq/OL9nlTFsZ87QAvsOO+CRZ1iePBU4Joqpq
nOhgAF5Q9s9ttkw3rwH0gD61xiiTGmTQIX4iD0wvrBeIZV79kVHvSiWUBwklM0cA
624X1LVO7PiXvBKXK7LroFq9v7/MgeViPPccn3NqT/cTaT2EJwv+rQ==
-----END RSA PRIVATE KEY-----`

// TestCertUnknown is a snake oil SSNTP Unknown role certificate
// for test automation.
const TestCertUnknown = `
-----BEGIN CERTIFICATE-----
MIIDCDCCAfCgAwIBAgIQeJh5KJ7Y2cFuI7RD+DnWPDANBgkqhkiG9w0BAQsFADAL
MQkwBwYDVQQKEwAwHhcNMTYwNTIwMTI0NzE1WhcNMTcwNTIwMTI0NzE1WjALMQkw
BwYDVQQKEwAwggEiMA0GCSqGSIb3DQEBAQUAA4IBDwAwggEKAoIBAQDfSGvAbitK
jzfqC+Vh/s6AuKYruFmOIJfrLaX0XYKWgDfVPqYP67ALCuFrzjp+vcJK8kj5QSM8
0T44DbmputD/w7aQEAbexQz5xlvqAhMfcZM1qkH3DY4V2LL0v/vgXSpvwVgImdSe
enaeNJzIWwGot16GyonG6r6oKIbT0pXzRBSfUxynhGGvu6oMp6p4Xe1mBtBrZTwh
k//JqWTvX/V+N8tlMo8Y6WJgYsGayQWzxQ19CsyfVFECOqSz2p+BLe4hFtOnC+Ro
LmaOECc8Mlh40V34hnvB6oobh/afbRmzt2y205l0Kf2R+567Yah4qFYYRd1TX9hf
dWCmasM4b4K1AgMBAAGjaDBmMA4GA1UdDwEB/wQEAwICpDAPBgNVHSUECDAGBgRV
HSUAMAwGA1UdEwEB/wQCMAAwNQYDVR0RBC4wLIIJbG9jYWxob3N0gR9jaWFvLWRl
dmVsQGxpc3RzLmNsZWFybGludXgub3JnMA0GCSqGSIb3DQEBCwUAA4IBAQAOX0Jn
+1ovjvJIOgH2POtLPyy2C+gS0j2XKWsqujjPaMv8LPgfwM0/C1DZ3nVYrbrh4k29
GMnISYp+pZ9iMJJ/xjCBVdd5GOJPWd97cNpYSNJ2J5eeVKsIW2UCMzNUQCSQ9UaW
ksB7m5wohk1FwnkAs45K7NaeZERZl7mTvlkt/X+gSUs/Jn0m7q5kAYqcVOucSa/8
SP07tciF7p7cLxgXYLOEgbh6TExd8ZIPkARCmGZzKE/NiK42wTACBQnyw/YMterR
mZGqdDkDhX5HrV/8dp0aitnDYNOrGQE+inDZ3eaA4Y5olkf0CIebFYBuBdTQtdhn
dsHwEx/eVHFDgCkh
-----END CERTIFICATE-----
-----BEGIN RSA PRIVATE KEY-----
MIIEpAIBAAKCAQEA30hrwG4rSo836gvlYf7OgLimK7hZjiCX6y2l9F2CloA31T6m
D+uwCwrha846fr3CSvJI+UEjPNE+OA25qbrQ/8O2kBAG3sUM+cZb6gITH3GTNapB
9w2OFdiy9L/74F0qb8FYCJnUnnp2njScyFsBqLdehsqJxuq+qCiG09KV80QUn1Mc
p4Rhr7uqDKeqeF3tZgbQa2U8IZP/yalk71/1fjfLZTKPGOliYGLBmskFs8UNfQrM
n1RRAjqks9qfgS3uIRbTpwvkaC5mjhAnPDJYeNFd+IZ7weqKG4f2n20Zs7dsttOZ
dCn9kfueu2GoeKhWGEXdU1/YX3VgpmrDOG+CtQIDAQABAoIBAC4Ktv1lOlQTmEoQ
zeoZKufsK4fUadMe8XSzYwcyyRzTr6+vTXr29MQyKC2cxv8fjhL6RwXu4p9F3nic
i+5t8W7sIpcuVtxU8coHlyh36Q2Q0/Q6+YNCLkxe9Pf3Jq8rRvbofYi4f9l2E0Oo
QbQm3/60YS+WUDbObSYrxTGkuPsCaiTNqlTr84w+GcUrc3QS5gt5eD8nlXocAwoB
FE5ZmBbFuc0ylzZhFVTikGO1d2gL0xjNKQl0k+fjCj9T78P4wwycumlla/m/yAnr
yhJWBCQkW+MrQXLEpzVLAWMwnjNUdyCiUPgQiPDtMDtbHzsIzi4I37IodfVSnc1A
c5yM04ECgYEA4hHebh7JL/o1gTU181vx8NkuMj43OZh3O6g906xTBNZp3HFrxAlF
kiv9sJCCeTn1siDEjxNbxK16fS/qf/4Y1CHqkS5e9HMQQRyG9jLS4fiJc/YnBkHS
0EtRXa7Rd+7WvDgpx5ctsdP9giDiCPQQutcVbZzOwsSMro4TFoHHmrECgYEA/NgY
mILHFOlcETbR0mX9sj9W3Iar0cgDjL1vY1qn41mviZaHtEXZnigygGvzNwNG+zE6
9Aj3WQ/HbRF6na0gaQpHI8fmT0IR82aguCNS1m1T/9dGLUm5MvnPClS0TsKq62zx
ZSjEqMOY7rZlNHw39STKyqjPmFsihvuLONZSIUUCgYBW11RhatQP6Qaanq0d0ckL
ovHa/QlLx6SttwAhCsZNSmwZ8Tvbb1BZSSrHo4trM/eMuIepCl+rGpS9+CUVi69P
9cNch7qUHos059d0Rau6gDWU5Q6ymaB4wSX9XcU7U/ULEmwCLrGv6OYuEaGinNa8
XxjtJVpLeSMtfogYkjvx0QKBgQCYIFCUGkM7wrgBJ3GQ3IqKn29cma5xNp1kJWoK
ZjYTJRfneWlGvqwTa24PNGQOWmtvoQwuXeKsdEDxz41tpweUC3oH/jMExuTUBJB4
mdAycW8TxGtVvkCuefzm41Xk+V0q7s5CpgfE3oJ6RcWYkZB9b1iQHIdizJp2iowW
c2TQ3QKBgQCAemLhJftZMhJOqOj4CFQnRCQ6fh/rG8Hfzxw8TMXEJqHzxapoqbgv
b1v7j/lNE5+Yx8rBJJrMRs0hub5TTxfnOp0X28dlDmHTXQrbJs7WygfFlnJDigCk
E8y6D+kRk5Kli/23SeFR9N7SNBXQEB43KsiKMAzSUCLmaigyi3d8LQ==
-----END RSA PRIVATE KEY-----`

// TestCertAgentNetAgent is a snake oil SSNTP AGENT|NETAGENT role
// certificate for test automation.
const TestCertAgentNetAgent = `
-----BEGIN CERTIFICATE-----
MIIDHzCCAgegAwIBAgIRAPBK5FLIveq6Rvnq4qBI190wDQYJKoZIhvcNAQELBQAw
CzEJMAcGA1UEChMAMB4XDTE3MDUyNjAwNTM1NVoXDTM3MDUyMTAwNTM1NVowCzEJ
MAcGA1UEChMAMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAt3dBCVjS
cXCb5ElpmMcYceQgdsYW2NK6j7pCU/hDDUeGdsMBSF6FeIn2kHbMneb6gJ/m5Dbb
RtEfZZ6JoR4BX6WKz5W7Y1D2+gMlMGluV9OOudXPIP6YpJOtQahe5viQ+9BTHokC
dW5dZ4NtDlk178oLlX7tmcPD2F6YaWXf7+sRRt+GB7U7GQ1km6ghaF32cz18/rm3
wGFUzenmoo70SaEur2cVmE1E68b1WY/681bTcHEbks1h3rVf9g34ghI/gGHKjUOC
NwIhwtLp2vmLlqRufcdSST841XAGXq0Dg0j2eM+04O5BPuJ0iEgSJxbgcMJhfCF8
nrZw4HHWddIK0wIDAQABo34wfDAOBgNVHQ8BAf8EBAMCAqQwJQYDVR0lBB4wHAYE
VR0lAAYJKwYBBAGCVwgBBgkrBgEEAYJXCAQwDAYDVR0TAQH/BAIwADA1BgNVHREE
LjAsgglsb2NhbGhvc3SBH2NpYW8tZGV2ZWxAbGlzdHMuY2xlYXJsaW51eC5vcmcw
DQYJKoZIhvcNAQELBQADggEBAHIaCBsPn5uJPm8eNDDNzhyx6MzO2BIS4XRDvhk4
rz3H7bXaPjkwgYqSvWNOGTslqQcO/XATJE/RITgb8Z3E7ZdC5mjswRXA77ybjUaG
xSr51yj5ZokXVgeMin8mnBkz+e5j2Zry7SVweg5oa1bD+NYul1gUqOjCme04L2az
kyM0vJSQ53D16fZ/Ms0kSB+Uy8UOYgyP+z6zY9PU1G6gN03lO8E21fp1AmXgcCU3
JIocPYfGFTv21NRfAEk4YsvXqwTsGY9QJrESeabLjbIVfY9uOykSdfTB8PVdaN4V
K2YCC/kLxAF08OTmBFfUhhaiARaR6lfuQrtZBP5ktuWSZxY=
-----END CERTIFICATE-----
-----BEGIN RSA PRIVATE KEY-----
MIIEpQIBAAKCAQEAt3dBCVjScXCb5ElpmMcYceQgdsYW2NK6j7pCU/hDDUeGdsMB
SF6FeIn2kHbMneb6gJ/m5DbbRtEfZZ6JoR4BX6WKz5W7Y1D2+gMlMGluV9OOudXP
IP6YpJOtQahe5viQ+9BTHokCdW5dZ4NtDlk178oLlX7tmcPD2F6YaWXf7+sRRt+G
B7U7GQ1km6ghaF32cz18/rm3wGFUzenmoo70SaEur2cVmE1E68b1WY/681bTcHEb
ks1h3rVf9g34ghI/gGHKjUOCNwIhwtLp2vmLlqRufcdSST841XAGXq0Dg0j2eM+0
4O5BPuJ0iEgSJxbgcMJhfCF8nrZw4HHWddIK0wIDAQABAoIBAQCvMYvDVzQt63CS
AKB7qbNoHoX5pZNvnVtVoHFyKeItrh1zDygMaWZfAq+WqDsglc6kQQ2E4/VknJG0
wj1+w15gbX4uWDdG0avmdlZu8l7SM01ZnBhc04IDtpl910A4oygfroGQ6RiV9rvu
+wuK5hmhT3jcWwadDNnZpgs4qnW7bUcxrOEy1V1pCR2UEiiYA7YFndLun2SxzR4y
bKG+ZDRlOPdEfP9FVBVwbVZ59wqWRO17SSPIxa6zRztbg7F7NEmKNcDYBgLHGJuD
ySnW8VV/6OFTmdTjKfEpK0MyhX9HR6WJxyeghxX+JrsDqvQ9Mvaqu9L14PYFirVg
yQ8s0/ZhAoGBAPDHnS83Go7T4tQkS3afGA8UMU/OfPBwzAw8L1llQQJBkhJu+9bG
Vdhsgw5n7SlL97xE5gdDAo8+BgvdwsDtKfyN2fF6OZ3DK3dEocseQOsw0D7kFmcu
eVI3RiemFIFG+K3r/S93tm0G+cKpq2CTseRholo7C2yrQ37U08TWfZdpAoGBAMMQ
KlkbVnb3tDH78ML+bwXO72iwXv7S9WvqCK6ffwm6DaK+5nTMgQmqVAgppZ1cqYAJ
ZGLBCAdq3mi5mXu0EhbbRBpN9OVfWNPStbLy1eGGE+3nHz2iFGHlk3sAIUBQ4r23
YvrtklpBVLDgxaZCrFV/d98xGa3IKshwAgtISOTbAoGBAKf83PHAJEtaEXupBu1v
+j0q/WyMyCaIzBQNOYvJVR3Z2av6usISBnrE2nsGjzSsx98WwtZ2Lib6QwWsZuBr
l0uZPGF5wREMxhqkS62HIgv1NpVqVScQCZ0O62dmPBAmEAJoD3E6uJBAuajS77ql
0QtiAv+pCkN7CdBHdKh0bZNhAoGBAJYwMCsDnYNkHV4O+cVpWdpDBpq4kavqigRY
4e5x58J5el5AVfjALOpgNutCBb4vxmJK2PwgXCo54p0HqmFQuEzY7orCUzj4PNB7
gGMUDhHixh16wtcVoFPwC6m84909ahdgx9kkancLrkWyCvyEgWQjDQzQJVFkuWwy
saA2O8nZAoGAMy0AXGLVqyMKtRLCD79hur67GJBYnYA7Jz7HeCOxZHP4Ap+LabFi
4TxH0ug/1sun8vc0qmUKKzGoCRef9jjEmedvBp3UmDOULB7G+QfnxowsO8jXUEn3
tij2VCLGaZNTL9SXcK5IqwJM+rCXrctqiglvpK5BXH/4H56KYvgSHNY=
-----END RSA PRIVATE KEY-----`

// TestCACertSchedulerMultiHomed is a snake oil SSNTP multihomed
// scheduler role Certificate Authority for test automation.
const TestCACertSchedulerMultiHomed = `
-----BEGIN CERTIFICATE-----
MIIDLjCCAhagAwIBAgIQEzJ+Q0fNrVJYOTxilkF03jANBgkqhkiG9w0BAQsFADAL
MQkwBwYDVQQKEwAwHhcNMTYwNTI1MTYxOTQxWhcNMTcwNTI1MTYxOTQxWjALMQkw
BwYDVQQKEwAwggEiMA0GCSqGSIb3DQEBAQUAA4IBDwAwggEKAoIBAQCuVa1wX7+1
7HGk2k82pnputYcYG7nRTn8AOSiAS6CkFbj7asm+HS2wAgOaIZ55zIPjAjaA6oBt
Jn0J2uM5W6lm+RyGjqufVvPjfuIKeHytfbBcZbAGfuZNkbnwc4VT31UtKY8P3pTJ
rsDI7vxA5CopKY7XtUaKDyex1eCzx2XA10L1PD5G7aWk7E/4yIX1nGt6CqZJqG2J
eok7bZr7RLC3Yr7BXgPQvN3xSrDNzks+7f0U+oqQwrB65qAdoOt8axjssA9mCBOI
bLamoKU6iSWzJAtWJ6YRfJSxefM8xSzkiqdhEqsU03ZhpKuD8KJ97u/hZXnMgwTP
d5gLeJnnticpAgMBAAGjgY0wgYowDgYDVR0PAQH/BAQDAgKkMBoGA1UdJQQTMBEG
BFUdJQAGCSsGAQQBglcIAjAPBgNVHRMBAf8EBTADAQH/MEsGA1UdEQREMEKCDmNs
ZWFybGludXgub3JngglpbnRlbC5jb22BH2NpYW8tZGV2ZWxAbGlzdHMuY2xlYXJs
aW51eC5vcmeHBMCoAAAwDQYJKoZIhvcNAQELBQADggEBAHfE2eZYjcrp7UJXoztV
BQyGHphYUJrCbljtNn7SYEqO/i6Lh4t97GNuJosJYEW2J2+Bu/zVVFRNO7SLPhNs
hLNKvalWarVfa5Rp5AgvAqxYvXkDUzL2ZrSminj3a24pd60BrrGnoF3vhQVN9UEe
hC2z173W/evzoPpsWCEeTAJUIPhiAthqFoc/PAGs0S2pxYwo/FVuyoB4OwcWp3VM
mzZXQFY+Z3Y/v3Hn9EBr08yjSps8hH2ZmEy7zOHEQT7aswKHe3WUt98MAHpx9IVS
6XdH1TF5KWFm2tBozhuK04EbwZyrF1Oa6oZqKw3YAURpy7+NF7gYXJ0xaGIU6kWV
AuA=
-----END CERTIFICATE-----`

// RoleToTestCert returns a string containing the testutil certificate
// matching the specified ssntp.Role
func RoleToTestCert(role ssntp.Role) string {
	switch role {
	case ssntp.SCHEDULER:
		return TestCertScheduler
	case ssntp.SERVER:
		return TestCertServer
	case ssntp.AGENT:
		return TestCertAgent
	case ssntp.Controller:
		return TestCertController
	case ssntp.CNCIAGENT:
		return TestCertCNCIAgent
	case ssntp.NETAGENT:
		return TestCertNetAgent
	case ssntp.AGENT | ssntp.NETAGENT:
		return TestCertAgentNetAgent
	}

	return TestCertUnknown
}

// RoleToTestCertPath returns a string containing the path to a test
// ca cert file, a string containing the path to the cert file for
// the given role
func RoleToTestCertPath(role ssntp.Role) (string, string, error) {
	certdir, err := os.Getwd()
	if err != nil {
		return "", "", errors.New("Unable to get current directory")
	}
	cacert := path.Join(certdir, "CAcert-localhost.pem")
	roleCert := ""

	switch role {
	case ssntp.SCHEDULER:
		roleCert = path.Join(certdir, "cert-Scheduler-localhost.pem")
	case ssntp.AGENT:
		roleCert = path.Join(certdir, "cert-CNAgent-localhost.pem")
	case ssntp.Controller:
		roleCert = path.Join(certdir, "cert-Controller-localhost.pem")
	case ssntp.CNCIAGENT:
		roleCert = path.Join(certdir, "cert-CNCIAgent-localhost.pem")
	case ssntp.NETAGENT:
		roleCert = path.Join(certdir, "cert-NetworkingAgent-localhost.pem")
	}

	if roleCert == "" {
		err = errors.New("No cert for role")
	}

	return cacert, roleCert, err
}

func makeTestCert(role ssntp.Role) (err error) {
	var caPath string
	var path string
	caPath, path, err = RoleToTestCertPath(role)
	if err != nil {
		return fmt.Errorf("Missing cert path: %v", err)
	}

	var template *x509.Certificate
	template, err = certs.CreateCertTemplate(role, "test", "test@test.test", []string{"localhost"}, []string{"127.0.0.1"})
	if err != nil {
		return fmt.Errorf("Unable to create cert template: %v", err)
	}

	var writer *os.File
	writer, err = os.Create(path)
	if err != nil {
		return fmt.Errorf("Unable to create cert file: %v", err)
	}
	defer func() {
		err1 := writer.Close()
		if err == nil && err1 != nil {
			err = fmt.Errorf("Unable to close cert file: %v", err1)
		}
	}()
	if role == ssntp.SCHEDULER {
		var caWriter *os.File
		caWriter, err = os.Create(caPath)
		if err != nil {
			return fmt.Errorf("Unable to create ca file: %v", err)
		}
		defer func() {
			err1 := caWriter.Close()
			if err == nil && err1 != nil {
				err = fmt.Errorf("Unable to close ca file: %v", err1)
			}
		}()
		err = certs.CreateAnchorCert(template, true, writer, caWriter)
		if err != nil {
			return fmt.Errorf("Unable to populate cert file: %v", err)
		}
	} else {
		var schedulerPath string
		_, schedulerPath, err = RoleToTestCertPath(ssntp.SCHEDULER)
		if err != nil {
			return fmt.Errorf("Unable to get scheduler cert path: %v", err)
		}
		var anchorCert []byte
		anchorCert, err = ioutil.ReadFile(schedulerPath)
		if err != nil {
			return fmt.Errorf("Unable to read scheduler cert file: %v", err)
		}
		err = certs.CreateCert(template, true, anchorCert, writer)
		if err != nil {
			return fmt.Errorf("Unable to populate cert file: %v", err)
		}
	}

	return nil
}

// MakeTestCerts will create test certificate files for all roles
// that return valid paths from RoleToTestCertPath
func MakeTestCerts() error {
	err := makeTestCert(ssntp.SCHEDULER)
	if err != nil {
		RemoveTestCerts()
		return fmt.Errorf("Error creating scheduler cert: %v", err)
	}
	err = makeTestCert(ssntp.AGENT)
	if err != nil {
		RemoveTestCerts()
		return fmt.Errorf("Error creating agent cert: %v", err)
	}
	err = makeTestCert(ssntp.Controller)
	if err != nil {
		RemoveTestCerts()
		return fmt.Errorf("Error creating controller cert: %v", err)
	}
	err = makeTestCert(ssntp.CNCIAGENT)
	if err != nil {
		RemoveTestCerts()
		return fmt.Errorf("Error creating cnci agent cert: %v", err)
	}
	err = makeTestCert(ssntp.NETAGENT)
	if err != nil {
		RemoveTestCerts()
		return fmt.Errorf("Error creating net agent cert: %v", err)
	}

	return nil
}

func removeTestCert(role ssntp.Role) {
	caPath, path, err := RoleToTestCertPath(role)
	if err != nil {
		return
	}
	os.Remove(caPath)
	os.Remove(path)
}

// RemoveTestCerts will remove test cert files created by MakeTestCerts
func RemoveTestCerts() {
	removeTestCert(ssntp.SCHEDULER)
	removeTestCert(ssntp.AGENT)
	removeTestCert(ssntp.Controller)
	removeTestCert(ssntp.CNCIAGENT)
	removeTestCert(ssntp.NETAGENT)
}
