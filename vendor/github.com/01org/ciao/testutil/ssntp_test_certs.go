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
MIIDFzCCAf+gAwIBAgIRAP+4vV/CO3EX1kfxO6jJJvAwDQYJKoZIhvcNAQELBQAw
CzEJMAcGA1UEChMAMB4XDTE2MDUxOTEyNTEzM1oXDTE3MDUxOTEyNTEzM1owCzEJ
MAcGA1UEChMAMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAzaf2jLHu
VVJBFTIkabC5eI9cCv8TazX26MuFYbHROdHXKycHTaj0q9hJlxNqIrNgWvkluQq1
ov37nW0RGn58LVwEN1RY+g+Q1EEURTJfuiwtgA2OdVXMjRC0gjiO3ertmDkpE673
5FfAGpxtO862M1h3PfImcWml4y8Sdg6Uxq1UT47mNCBUHCf9ZASsP+U3DM737w/d
qydadn6uM9+BtmdRPo+jxoI7pTLWa2l5J/CQSNqHl5g5NkO6hO8V+2c7RFgpWGL0
XDGt75x2e1BckGERM4hno27FVrC2WpceZrG8HcQPR8RQFq29Re+Ewc9NOpNKA5R5
HqBcJdmxa9GnyQIDAQABo3YwdDAOBgNVHQ8BAf8EBAMCAqQwGgYDVR0lBBMwEQYE
VR0lAAYJKwYBBAGCVwgCMA8GA1UdEwEB/wQFMAMBAf8wNQYDVR0RBC4wLIIJbG9j
YWxob3N0gR9jaWFvLWRldmVsQGxpc3RzLmNsZWFybGludXgub3JnMA0GCSqGSIb3
DQEBCwUAA4IBAQAEZfk7cx7Cu3pOVO/p7FgtV/gU2bTopGEg0MurHL9670Fs2vzs
Gm0PWtaKFkjP/iYOdDv3ffuOPlPmH5s+WQSBTOQpte2WzxIOYl10kOvbV3zrpCuA
okn63SXAhKHwuSxvRed5WDd22iTJKUThr2EV3zk7oVe4GPFNOOXRQiVwg8tIDdSx
KeNlW7SKTYRD7utRJRuVzTkO81fz6kw5JoqIXIFipeqrPm5wVBD6z1tSIIKqV8DO
AhUd9Gk2Fb9V/YQ9UWp9tl+hEzVrpBuYPXqrMXHif55Fo5JoTMeZeIgHPLZ9/Utc
jlsGuomSwavQtm+V18VUt3U3XIghFeqc5OVL
-----END CERTIFICATE-----`

// TestCertScheduler is a snake oil SSNTP Scheduler role certificate
// for test automation.
const TestCertScheduler = `
-----BEGIN CERTIFICATE-----
MIIDFzCCAf+gAwIBAgIRAP+4vV/CO3EX1kfxO6jJJvAwDQYJKoZIhvcNAQELBQAw
CzEJMAcGA1UEChMAMB4XDTE2MDUxOTEyNTEzM1oXDTE3MDUxOTEyNTEzM1owCzEJ
MAcGA1UEChMAMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAzaf2jLHu
VVJBFTIkabC5eI9cCv8TazX26MuFYbHROdHXKycHTaj0q9hJlxNqIrNgWvkluQq1
ov37nW0RGn58LVwEN1RY+g+Q1EEURTJfuiwtgA2OdVXMjRC0gjiO3ertmDkpE673
5FfAGpxtO862M1h3PfImcWml4y8Sdg6Uxq1UT47mNCBUHCf9ZASsP+U3DM737w/d
qydadn6uM9+BtmdRPo+jxoI7pTLWa2l5J/CQSNqHl5g5NkO6hO8V+2c7RFgpWGL0
XDGt75x2e1BckGERM4hno27FVrC2WpceZrG8HcQPR8RQFq29Re+Ewc9NOpNKA5R5
HqBcJdmxa9GnyQIDAQABo3YwdDAOBgNVHQ8BAf8EBAMCAqQwGgYDVR0lBBMwEQYE
VR0lAAYJKwYBBAGCVwgCMA8GA1UdEwEB/wQFMAMBAf8wNQYDVR0RBC4wLIIJbG9j
YWxob3N0gR9jaWFvLWRldmVsQGxpc3RzLmNsZWFybGludXgub3JnMA0GCSqGSIb3
DQEBCwUAA4IBAQAEZfk7cx7Cu3pOVO/p7FgtV/gU2bTopGEg0MurHL9670Fs2vzs
Gm0PWtaKFkjP/iYOdDv3ffuOPlPmH5s+WQSBTOQpte2WzxIOYl10kOvbV3zrpCuA
okn63SXAhKHwuSxvRed5WDd22iTJKUThr2EV3zk7oVe4GPFNOOXRQiVwg8tIDdSx
KeNlW7SKTYRD7utRJRuVzTkO81fz6kw5JoqIXIFipeqrPm5wVBD6z1tSIIKqV8DO
AhUd9Gk2Fb9V/YQ9UWp9tl+hEzVrpBuYPXqrMXHif55Fo5JoTMeZeIgHPLZ9/Utc
jlsGuomSwavQtm+V18VUt3U3XIghFeqc5OVL
-----END CERTIFICATE-----
-----BEGIN RSA PRIVATE KEY-----
MIIEpQIBAAKCAQEAzaf2jLHuVVJBFTIkabC5eI9cCv8TazX26MuFYbHROdHXKycH
Taj0q9hJlxNqIrNgWvkluQq1ov37nW0RGn58LVwEN1RY+g+Q1EEURTJfuiwtgA2O
dVXMjRC0gjiO3ertmDkpE6735FfAGpxtO862M1h3PfImcWml4y8Sdg6Uxq1UT47m
NCBUHCf9ZASsP+U3DM737w/dqydadn6uM9+BtmdRPo+jxoI7pTLWa2l5J/CQSNqH
l5g5NkO6hO8V+2c7RFgpWGL0XDGt75x2e1BckGERM4hno27FVrC2WpceZrG8HcQP
R8RQFq29Re+Ewc9NOpNKA5R5HqBcJdmxa9GnyQIDAQABAoIBAEML/U9FOwRJ+rnk
TQa//NeXNVTIcBZF06d1opiFFkcZaGLDKJhi+tGDhApi4/lILaO7EldPbIQk9YEP
a8INdoJ7O0ymjROJO5hXVzpv/9F8UaErykPqovNifNbvhXRIAQndqMyoAF1LVm/W
i64x6Ci5MLbbWTkkTlbQo94lRs+2YLVFwQi13DewmSgE79P51PWlXTZcDRHQeUbM
hZILmct0EQROW0qKKe+qLNy2X9yvGlI82JiQqXzP3MDc463eyicCqwF3KqGmiKGe
FaVe/fJFKWi+Jo1HYC8ghoIeho8nlhU+kiWvK6K0yrD1B6/hyXrxnF7UhQCqnTMo
Q+E6vWECgYEA2fZxeoz0r3CQ5jCt8t/kJBgpcyx7teghPrfQRpAlVQP549ALkmP2
BkcK7VIlLwxWBlTv2eagvS57SE6irpkIBBWPicP+xMf8VF9oqI3dUutHWN03opUC
BlSia2mL9HundztMAjKzcPqGELu6xAkGEkl9vKIkYo1NHGH0JM0Ft50CgYEA8Yu4
SqyHDVjuVLT6qK+Evb+Wgd0jjv9SLcYi4tP/0FqOQGZyoKB5s/3xH8W3Qi/v3DPo
obFVx4uKgZZ9z0ex9ei6pw36pnmef1oDQ3ldDCt/VA1H1wL+ws/qhRfBhDEVRwHm
pigmHHUgKB8kL7ueuhsy1/kdarBKVhoNGayj1x0CgYEAk59BXnJHaud/jBheSAfx
uayPrkzrgNm2YocWTiRk1H676drHa5++SqQlN3UScBoXzXQLevaj2V2469Euh7hn
4HRF4lXXoKmeMfropHho9Tca/Infm4L2exkpZDx5KN3zH2MO6NI0DInw8TZkmU9P
SGVz/qWGpST0nAuP0rj1bVUCgYEAyDrRZ52DSo446yEnVGRDPmQuaLKfQm/meKlx
y+R/gAFBQKNsTDkbChjtJDBrDMPeKwUgx8DQYd0L0QamfghpvnbRG6Bb8lqJB/rf
D7TwbCE2qL9lmRgThfyC5RdfRKzHfZhW0dAgX6C8KmY+Qg9esdnQGPaZ6xH+XUe1
Kl5RZwUCgYEAl2g/UM0T2d36AlLiijanLBSsDgni/kOhYV+6tuflYJSbjihlX67D
ObDxa2XeZP1bWAvDzngY2XY/O43eY1fyRocp0mzTnFjGxB1tzZZNrMhXGy/6eL1p
P4VdmGGSMY/EUowzeKqktHRvBJDMGKxfaxTivlfyUNGQ3Bo/7UoxSU4=
-----END RSA PRIVATE KEY-----`

// TestCertServer is a snake oil SSNTP Server role certificate
// for test automation.
const TestCertServer = `
-----BEGIN CERTIFICATE-----
MIIDFDCCAfygAwIBAgIRALpy2XcRn5fAkBK+D0BR/FUwDQYJKoZIhvcNAQELBQAw
CzEJMAcGA1UEChMAMB4XDTE2MDUyMDEyNTEyNloXDTE3MDUyMDEyNTEyNlowCzEJ
MAcGA1UEChMAMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAz5m91YXi
B+HVGWlstLMRe2gEeB/AEObD/WQdpymmcI6JxQJCtC19M+mOcz+UGxGk01QZtccK
yykRNr4RlgbSP1kpiBEUwyY2zAM0lXwBT+qKo/Hd4vb8BsQejohGvfV7fdJMZ29/
4z0k+RVVGuWc7JjMKrSPEFmv351CWo20aYCU/JHC2w26/akHdnFKr0WAChfJHMH0
0fgqvmmE/QRJr8zFFHg1BZdIbWiUm3yTkyC24V/3tThJ55rUr1WRpBwjG+2/hJsr
2dJRa4EXmEglU8vJWWb/g7nq4qzzznePHBOnFp+wjBB+UaXp3PXYnysSp/gZbhg3
Dmt4r3cabQFOIwIDAQABo3MwcTAOBgNVHQ8BAf8EBAMCAqQwGgYDVR0lBBMwEQYE
VR0lAAYJKwYBBAGCVwgFMAwGA1UdEwEB/wQCMAAwNQYDVR0RBC4wLIIJbG9jYWxo
b3N0gR9jaWFvLWRldmVsQGxpc3RzLmNsZWFybGludXgub3JnMA0GCSqGSIb3DQEB
CwUAA4IBAQCbxNrtd1M6hNk14hEUPNqnNRJMcstK5p6eVR1okP04LV3NkVQn3h8e
F6fPZ5NdvPaaKgwg4xZUnH1sa1w7+QwiZCwUF7AuaPOEepmpFlorSWJyXMnUakgP
b6Io/ZsB3i3niiKlPpKGr1Vzd98MKjJX5hbDpdPRg8qZBulXJgKoLKQSIQzWQB45
Qtyjjb2e1ShildDTGngik4epWkIldB95SNdVrpiPBy2EcG2cgtOEhmY1IlVV3N4v
8Qi7E7SwHL62LNqK6Alb07LRhWmG58ik5xFEVoXq6bpodllCpdaywwuZ7baEGvCw
CgaiyxWLxuSJ63X6n0CQNcvxxtTpLPUX
-----END CERTIFICATE-----
-----BEGIN RSA PRIVATE KEY-----
MIIEogIBAAKCAQEAz5m91YXiB+HVGWlstLMRe2gEeB/AEObD/WQdpymmcI6JxQJC
tC19M+mOcz+UGxGk01QZtccKyykRNr4RlgbSP1kpiBEUwyY2zAM0lXwBT+qKo/Hd
4vb8BsQejohGvfV7fdJMZ29/4z0k+RVVGuWc7JjMKrSPEFmv351CWo20aYCU/JHC
2w26/akHdnFKr0WAChfJHMH00fgqvmmE/QRJr8zFFHg1BZdIbWiUm3yTkyC24V/3
tThJ55rUr1WRpBwjG+2/hJsr2dJRa4EXmEglU8vJWWb/g7nq4qzzznePHBOnFp+w
jBB+UaXp3PXYnysSp/gZbhg3Dmt4r3cabQFOIwIDAQABAoIBAHOZK2DTfUpgUTYm
QzbXo3txL1Poch23MhlN/0kO4zQ32rVODfCgh+A5RG4eUA1GpN5cLTjQTc1U39X4
vngo8jf+ISc4Q7Rq+gZeHpDCjUR/2JVz39c7Kpll6ZH6hlHOeOZWDN9n8fGKIaVl
YI9qnhgM+VsqUaOMHWfJ/KHJ2FUKGKSmdCR1dJBOtWcg5z9EkKmkdIDL58gg3029
QgopdSFqfc+qjnxuKgrpVHqrgirRbftFrymWqRZRA1PIgnngRl2McTle1rtGh2JR
rMi83aNjWJtMsw0HxAz0ERb1RixtKNnu9NUJ54X7UwvfQuPS0fWV8aoYHjQzoPz2
5pHfIlECgYEA4pAhAuwIu3jZuBtRzNCOdK9cKAf4r3R/GGXXH2CgguJytQkK5lCT
estkigjYk8MrEueyXYraEGDvWvFj3zIglUp+IKaS/RUZbEoA7DOZvdduf81DJ9Wl
09OHF8+Lx9SNkjA3sIYyGuVMoKkfSQwUtoxSpo4lbGK+q3nFeO1i/usCgYEA6pLj
ovZPnCcAsqtIIluHKstdmwaDxp/EfMRVQV5EnubG6BCWZLwC5R5j0KDoyG0gnbhQ
dd4k0qhZLXRtd/qqXf9vXNQ5eDx3m8Erbx7iUBwh0+vxO8eQvzDRvBH1EMADBk7e
XdeXoXVEFhbptq0CjQOXpfxzueGg/nOBE3PVz6kCgYBqpGzlczSpCblxb2qRfZmQ
UvqN3TKxY6RvV4BqxJDJCs3zaM44mrTQl+w7DO7knnkn7OeIIFOEYhxIMldQN8ge
fXHg7IdDmSreTfchNyims0DP50408ducWXS9QHQLG4GHzipobMIo5sWq2fBf8c/O
HT7KJx52ZgRZsnfA8/wlAQKBgAbocH+6FToaA/E/Dg7E90QRXR5VoMfWqKir937H
UeoEDdODuYoZ51PsAzB/rJtKa215ohT2h8sUXhvyk862uRGvlg37yf16emNB2w+Y
rz2AtpZRGneSNvcKbwLE3JyzquMiq3XEBZkhpPGplxRkH/EbK/odZyMQO/eCW0jB
XoiRAoGAdd028Hn7SIvD59Q0bDhsqweGAwiORh8gz2+KmG2Y9ePa3L1dNArOrOCH
itgD7Q1oMEGS2OA3l6XhdI/GxxXhxA2kXf0Q0vLq76RXHOpFmugdDbcLZABmjhUW
b4BEfHCZNDdcAGJ28MgfLlIuvGJIREM5wfRiKQFfXH2kvV8ua74=
-----END RSA PRIVATE KEY-----`

// TestCertAgent is a snake oil SSNTP Agent role certificate
// for test automation.
const TestCertAgent = `
-----BEGIN CERTIFICATE-----
MIIDFDCCAfygAwIBAgIRALFQMUl+gspsJnYGQTXl6lowDQYJKoZIhvcNAQELBQAw
CzEJMAcGA1UEChMAMB4XDTE2MDUxOTEyNTIwOFoXDTE3MDUxOTEyNTIwOFowCzEJ
MAcGA1UEChMAMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAop9DnL+m
9c+iGeM+AvzEOOr72Z7j26YrM5cGTpNBcPsf3foK1RYjcOeA1sVuc5ndGZXsGLtu
fBNKQCiqZuQRbFHroji8a8ooHnfR0Lzr7kfqLw48uJBX6tqMMUmnq2KeCHjXss/G
tGkgE27cGOCAbN8EDpQPDgdhvA381oK0V7nFq/UHB+AUN3v6+D0F37DbtDlip/FY
Sz3FecoqZh5z23TYpdYLkZcVhOZlnseV1L12kXa0CD3zJuUY5qGR2Q1xTvRScNrj
V0mJeOoowcN1dP/U9CLYAX7d+sI7LFvUe/jZzmpVvB6KexsHbNy1Ee763NwZzZi2
X9796fyh8uwfewIDAQABo3MwcTAOBgNVHQ8BAf8EBAMCAqQwGgYDVR0lBBMwEQYE
VR0lAAYJKwYBBAGCVwgBMAwGA1UdEwEB/wQCMAAwNQYDVR0RBC4wLIIJbG9jYWxo
b3N0gR9jaWFvLWRldmVsQGxpc3RzLmNsZWFybGludXgub3JnMA0GCSqGSIb3DQEB
CwUAA4IBAQAcSTLbJ6Yf5ch/b/RqKzcJ0CeNa9xJ+b6TRYsEUEG+feceodEkR/pg
+OdltB/skyd4Lz4jbIHgza+x5uPdMD7yM0iN5Wq2Z2JvnTrfKCq4dMGiXNan94Df
/+AIK6NWixBqFpolsjf6JfLh8Tvb1iinhcJrG9KE5k79pAw/HNgLiJ6RGZkHhRuM
mu7GxNL+0ctcMIW/04obQQwoHK6gCTQ6VDVT0yiy2yz4eI0cWElsoFNjiuxT46DD
5g7Fjju+eV9TwHahkBdu4UVkP/hw+hyrrErvEecfW/3l5gR6BxJDA8k0braP2Kh3
mGuazAuLvYfMr9Dr/BNFU7IE/c+bLpTU
-----END CERTIFICATE-----
-----BEGIN RSA PRIVATE KEY-----
MIIEowIBAAKCAQEAop9DnL+m9c+iGeM+AvzEOOr72Z7j26YrM5cGTpNBcPsf3foK
1RYjcOeA1sVuc5ndGZXsGLtufBNKQCiqZuQRbFHroji8a8ooHnfR0Lzr7kfqLw48
uJBX6tqMMUmnq2KeCHjXss/GtGkgE27cGOCAbN8EDpQPDgdhvA381oK0V7nFq/UH
B+AUN3v6+D0F37DbtDlip/FYSz3FecoqZh5z23TYpdYLkZcVhOZlnseV1L12kXa0
CD3zJuUY5qGR2Q1xTvRScNrjV0mJeOoowcN1dP/U9CLYAX7d+sI7LFvUe/jZzmpV
vB6KexsHbNy1Ee763NwZzZi2X9796fyh8uwfewIDAQABAoIBAEhDQUYsG8LrKvsZ
8XpeW8t3D8baRiJaqqPYHmNYKCJOVRDMhXe+yKzpEmVdggE4g/lUl389+pCD+eCc
sWvbOKrLlEuXrpKvWDiBwehhqu1NY5DZYL4a1hZ0Wwuj0S/lOJhHKoI4tfGBLVG1
V3RufmLijujzfeUb/qAUDyA7IGxCWSUVpKU29MzAHAXftm3ucN5J6bDPHaKe0KA7
ZddSZd2C1/i0Ev3Z4Z+/1hi7C7QZYJ35OoQv8+NSg7nBHatt3EPFtxaN5TOGgmfs
IE2jb2kqYaDU0rYp5IitgGa/zQI4q4RqT4q7oQrUmwyorEwj8Mj0u7C+9oUpeQir
teP+KiECgYEA2AQ+ybr35UE4PnluCu/gqr6xincvn6I8HqDIPQ13cM0eKA2Z5ROu
GTlP4bFONufKG53OG3bD5tfaW6fKLrY+7Tw1Q8qok3pwLqKqBXzcxqSmCD5jcUnF
9xstdVTz/YkZmBrBUQx40zlojgNIg1AQxVuXzsfHQ/cqzTmTTiYinmMCgYEAwLj4
Pd+Aqkd21o0ENqVL4rdfy3APQCSvsdyliYgMmnLYHBKVCyvZ1cuwe/DW86cOSIzK
0oDkv264Qa6vhKQV4XfEo/bMOFOYVxIcISTV1LIzabMSQ/fw7VIWVjvT5Vj4EOeG
vD4LelVtJxGI3Ri8E2O3jL7xSv2m5aa3t47vmgkCgYEAxqwo3zphUm2IgBUIe3Ch
jggym6oAl+4LIxQ29cfT6WANc0MHHmPaRIKskGOVDvRhssKRVDsH8+DkiFWqowmk
mGY+iunx3ynF0W5ztvZeyyeVOJHAvene8+UACyCmArG8Y2OAFr3ExmfPXIVyhKr3
sbwKw/iDsWO67uMcMszqHAcCgYAQpOjrjw3xptJgnTUr8wKmxeeEDl2C0KhL2B6D
zjgobpqzcfdlS5g4mqrXSWmHCXp2UZKCs5cN4WYQZiHKdtFc85cMAhiJFM8nVe0P
/7pn8Cv4iPqe3B72oAxFzkzylch2zUgZodIj8pTGtwD291fm5MnJYgQ80cNEOi3L
sJCI4QKBgH7BlfzRC3vC3pcvxMF3NbSNfBvH8aS/kcfuEjlLMvZoUDEW8Je1YHds
As96gP0LdkPCPDUGjjxWvw6IVd7v4Gadv6z8olNS4PZshDusLz4NBet990ZgZ3vr
nPFs9Q3f0Ck/Cblt6XyXAlEUBrS3bXDyO+vsrgpXVqzQ8xhNYT+w
-----END RSA PRIVATE KEY-----`

// TestCertController is a snake oil SSNTP Controller role
// certificate for test automation.
const TestCertController = `
-----BEGIN CERTIFICATE-----
MIIDEzCCAfugAwIBAgIQLj/lv0z3zgYtBuILQsFRMjANBgkqhkiG9w0BAQsFADAL
MQkwBwYDVQQKEwAwHhcNMTYwNTIwMTE1NjAyWhcNMTcwNTIwMTE1NjAyWjALMQkw
BwYDVQQKEwAwggEiMA0GCSqGSIb3DQEBAQUAA4IBDwAwggEKAoIBAQC/L10CLqSm
b+ZfPUEVerbbDUnyUfJ7DBZukVJTUS3ZT9qPSCbK3cqe3RLFJZMvfIxNdZZgbMJP
Xc+idH3xlJkaXhpu6oanyUMurJ2gDovkDE5JM1mfSVI9SuKxsTWmSoHOPN856+iC
dg9vrI/BcdSH4mcsMaUKY3LxIb6wEqrzot+dT+GepVx48t47Xwwna2c3H9OLSVZX
z+qIQwtb8oiH0P6BrzvyaKtNZL5EmotYqqgOKXKPetY2AQ/QQsC0y/iWa/Cfj6lh
VcMQaWs1h75CIAT5Mt/gtSP9csCrt/inhlj4o3GTwG/MhY160OIVSKtYWDasAqWy
bP0HcToOoWgdAgMBAAGjczBxMA4GA1UdDwEB/wQEAwICpDAaBgNVHSUEEzARBgRV
HSUABgkrBgEEAYJXCAMwDAYDVR0TAQH/BAIwADA1BgNVHREELjAsgglsb2NhbGhv
c3SBH2NpYW8tZGV2ZWxAbGlzdHMuY2xlYXJsaW51eC5vcmcwDQYJKoZIhvcNAQEL
BQADggEBAKW4Kso/RoBx7N/F+7F29/QCfMpKkWQ9X2h+Cf4pMCO2gsExXv06I4Sn
zNkRd8XX6LeuvgdHXOV+qVsx8+2yynvGQbLaaetcK0xBDKD5j3u0ljKlXyOkYi7R
o4NA50VtoYXg2JZPF2l16RhooIhcbtoryT2XadI6lAi4LPK9Mbsy4zofrmCR7hTA
usiae6Bn9Ff5TEtJpGYxVST1E7uk/ukYjgciaUgqGAxzytDZlmj5vc69qtOiCkD9
17XSkNMzXZyj4VZxF/65QyfEqBHas8UNm80yk+SnLGu1X8sS2U+d/uyMD+eqC1yc
ggxzGWcs00dfS6hL8/y+XlWOyymprQ4=
-----END CERTIFICATE-----
-----BEGIN RSA PRIVATE KEY-----
MIIEpQIBAAKCAQEAvy9dAi6kpm/mXz1BFXq22w1J8lHyewwWbpFSU1Et2U/aj0gm
yt3Knt0SxSWTL3yMTXWWYGzCT13PonR98ZSZGl4abuqGp8lDLqydoA6L5AxOSTNZ
n0lSPUrisbE1pkqBzjzfOevognYPb6yPwXHUh+JnLDGlCmNy8SG+sBKq86LfnU/h
nqVcePLeO18MJ2tnNx/Ti0lWV8/qiEMLW/KIh9D+ga878mirTWS+RJqLWKqoDily
j3rWNgEP0ELAtMv4lmvwn4+pYVXDEGlrNYe+QiAE+TLf4LUj/XLAq7f4p4ZY+KNx
k8BvzIWNetDiFUirWFg2rAKlsmz9B3E6DqFoHQIDAQABAoIBABBW2usR026KB7VC
BerxBumntBcqm7+aY9xlPRTzzihRY8t1DiOuWt/C4xTIRlD7ov4Hu6dYBC9GRDWN
ISphWchgHIA4OPPkBoLZq8r/E0OVLaeh5NnxKT8lxEQNchlZKsjWePl5SPDFaEJS
DCMrOE+4sLqdL464ux0Sljp0DfouXnlC7eDA20+M4FOBIp9zUZuwF+OvfPEASWml
kOCWgZHxpHmCv6wJRODEUdGqGqjq9mSoO6yfvFdaFQbFDMvUcgMZzL8y49dCP9N8
xY4WXcEnyCCAVz2RniGttUrJ+yz/Oku1e/poHcpUmiwBxNBSNFBYR+NLmyQ+CHH5
RetPSP0CgYEA1XNyTB5vbZPvQcxBQzYwcM56f1SfFnLmzZSZ04cQiT/wniQHtR2n
jUtit1Kh22uK6tkYwxGCy1KWvJWTvKQDf6c1x9ULg8xJkKWWEBABFA7Nns7mnOAC
g9Xjw8mMy6orTX3VdX2TU30/HFytuOjpBXGDUYaCmeARaNL7IxwJwPMCgYEA5Uur
cCCybwk0bkyWTmCeapfram/Tfloi6ZTGrLHyH82Oon7fXwfm9z9WJTQQ5YneF0QH
NY3+NuY60bRw94B6Unh5xAJXCHXq60PPmy/zXHnpz4FBWYo75RDZlOEa3Dpa2UBm
+je4Xj5BKaa1v6I/t1YZTnkHRUf/UALWWgR/9q8CgYEAmkZ+7hVxZDnwTBZddT7N
dDtIvo9jDM6vkxc8t25/vTPBrgtMptNwLue1ydqnsffgyC1xgEw/xMVEvbk+trG0
9abdcDnDwNb+tNV5yNJIdT7dz1Kry+b86lzF6tTaNrof4jp49hp1SXrVCqLRzTxK
b+zDhUE7VAxniOQ1MAMr6ZECgYEApDOwLbf8j+9zkJlf+fjO+V6Zw7sZJZ6+6a8V
J4626Xd28X8RzygFioHc2v+SKg608MxSrVNl/UKaVJp3W4ayEmUcLfXPBcwL0zbY
cTXBfTQA8AyME+ceRUfvyOH7LkLL1FB+bimA6lyCpaUw+m7iWhRaQwwA3OhWOaIO
hqA6UxkCgYEAlj72e1mdmGSgnNSlSRL5M2a16zDjme2jEv+6kJt61oniPkyH8wDq
jMy3pNuv6YJw8kd6wfDIn6G2axcJ3KZVo0Fg+kvIwcaUFHntCENBc+8eqR0IZV2k
cwf5P/DR8p7NHn92HOvnAzBYHvuxlRBibrQsLWgATT99sgSU1+/AKwM=
-----END RSA PRIVATE KEY-----`

// TestCertNetAgent is a snake oil SSNTP NetAgent role certificate
// for test automation.
const TestCertNetAgent = `
-----BEGIN CERTIFICATE-----
MIIDFDCCAfygAwIBAgIRAMv1c1CPD4YjMjsPzByml+0wDQYJKoZIhvcNAQELBQAw
CzEJMAcGA1UEChMAMB4XDTE2MDUyMDEyNDkzMloXDTE3MDUyMDEyNDkzMlowCzEJ
MAcGA1UEChMAMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA6wOK7Orr
bseBaxAMzbFDL1OFN/TrpJl5k/jOa0cd0IhhVG8cotYnUvND+ztfUXNfUYrhPYo+
dnJ00v6xXA7cBTyJuUh91uAwQ7xMt2TDOHFzgwNcfjQB46X1v4uL0aNspUUdUwjK
D5Ku2UB9ved+P7Pv/oLPxTCpz+0kIyEQhBOq5b3l/9G3KLeFYissVaJLvcZyY1rG
aOBPrQYxyD/Mv6dG6A4XbXfRIHnblWyLPLr/wwfRAjo7n/txrACc1nk8fnm9EYtp
bvFnrO89/GeCm52A6YutTAi9kiRHhCuK0L0qk5jtDgC8kowDJ4Eez1yvJ6bgBG3G
AIG0fJGO1EfEQQIDAQABo3MwcTAOBgNVHQ8BAf8EBAMCAqQwGgYDVR0lBBMwEQYE
VR0lAAYJKwYBBAGCVwgEMAwGA1UdEwEB/wQCMAAwNQYDVR0RBC4wLIIJbG9jYWxo
b3N0gR9jaWFvLWRldmVsQGxpc3RzLmNsZWFybGludXgub3JnMA0GCSqGSIb3DQEB
CwUAA4IBAQAqv7LN+BdG7Y4F9anA7/jFTpzOOLlSXDzvZ37aUqoGFecgIrb3YR2n
GUhdQK06zaec5qWvyNQTJZUwGXy/nbp3EpUvn05WH7A1VtcHduuyIp0k2TR+JVTa
KAGbhKsdGPy/5iCj6FWcstrLf2UAVc5wUAOL5PCcGgLZaRaNb+zAuYoF7V1f2DJ3
KLtZqhZfunzrmcrf/5NTng/LCXWrMjQbX8hkrKuzg44lKYnjuTbxwB69LkrPWrfc
sa1Se+W7gKi1vPS6oy48Wsr2B0bglUZ0vBzArSpJGA+jAfIzDmEn3yC3rvI6C9at
tJVh54fFCSkwpRtgdyjRE+EKG+NyCAYH
-----END CERTIFICATE-----
-----BEGIN RSA PRIVATE KEY-----
MIIEpAIBAAKCAQEA6wOK7OrrbseBaxAMzbFDL1OFN/TrpJl5k/jOa0cd0IhhVG8c
otYnUvND+ztfUXNfUYrhPYo+dnJ00v6xXA7cBTyJuUh91uAwQ7xMt2TDOHFzgwNc
fjQB46X1v4uL0aNspUUdUwjKD5Ku2UB9ved+P7Pv/oLPxTCpz+0kIyEQhBOq5b3l
/9G3KLeFYissVaJLvcZyY1rGaOBPrQYxyD/Mv6dG6A4XbXfRIHnblWyLPLr/wwfR
Ajo7n/txrACc1nk8fnm9EYtpbvFnrO89/GeCm52A6YutTAi9kiRHhCuK0L0qk5jt
DgC8kowDJ4Eez1yvJ6bgBG3GAIG0fJGO1EfEQQIDAQABAoIBAQCxoKXaV9zGiCg7
QZBLz5UWKixglM+eQxnvS3jJAKF6Qfo+lRSxxudF/PP+6Wsr5uW+fhesKdb4M540
86geCmUl2BHIZxAl3qDcMXBSlOgwux8xgNLh2HEtHPzXX6O4OaseZ1S4s8X1a0qY
jfP8GwIDJ/9XAIwFYLiYnYZYvt7602enNG3zcz4VsHZQh3NbG3cDsT8UbxV+qnAW
cpKJqoW+KOvwC+F52UbgKKAY6UjAHRZkeFeUZvbZ8VwHwxdUrADdwVFFXwvSWfe/
jcpz4FYFInf6/8ysIn0leRPy9JG1FFigYZtaOFtnoaAsFzXUv41WNvj9lm5kwYk5
RHx0WLChAoGBAP6dYCGV7A7S30rXgUpCiBjcvuBqo3EwvhoYDrbZbW3p/gecaG7j
mezuDmlCDJIGBlVFt4n0+WEqNCs+gwBQrmKXgx8vGL9XFZhyPEUYAwJe1X8bpI05
lWsnUun6gHKkTReHFdOmFPHQo36qmSf8WpmL/3theFq2zX+FfHZxO609AoGBAOxK
3gO2yG3GciL9iFJVrKdVID0x0C6KwJbNBw/zWps8k5o6xNCFWIyG7qJPMR7h5Z3B
5tYaUM5h7nsgequVed5OhaqS8xOulYrl9DHI+f7xZVdJ3h0bGzf/brfofb7ZSsMC
yM8RYwNj6ibU9EZvVorK0SVK6AtB2UNEOVX8titVAoGBALc8Er6Y4jUY1NFLniQP
FVqvIj7m/5Cp/2VQAubcOsBrMQHRMeb7rP6xo3Vkrx83br9XWOrTfdTLRpgIeMZ1
ScpyN07t2eV5inUXYQBoc2H1VbgP8LAhzMI8npL8UAww6boQ4UhbsZ8FA2RY6be5
CIqQEeB9GNxPyjwHmLa0broRAoGAA7axgpFu2PrTdGVTrSeXjRGzbgLIaNLZcAVM
5R1IAUSUdUoTKcvOtnawbXCdLwUR3MbdX+QN/RBg9SJvix7QSYQmaaXhmB+YThSL
H/UuqKkWlKaejQqOBPVIwi8vOr6jhCkZCtgVHEqHtZCHPkwlqgzB+LoSp4qjZYE+
/XD5U/kCgYAE+Cjf7J3LrNaTCP4xFTnTn78LU2DC6lqlUkvaNbf5DPcukzj/ROly
2GXr8MLdUgkvRSBtiUZCL7l+tA6bmBvWD96DvCDCQQH0qMsBDk4ecGzFq8y3JPlc
wv2lAUxo7ePQK1f4g88nRmGpHubAhqk2xYZ02Mka1c5L71ZUKqhM0Q==
-----END RSA PRIVATE KEY-----`

// TestCertCNCIAgent is a snake oil SSNTP CNCIAgent role certificate
// for test automation.
const TestCertCNCIAgent = `
-----BEGIN CERTIFICATE-----
MIIDFDCCAfygAwIBAgIRAIYATy+4q5KBXFu+uNNJRUkwDQYJKoZIhvcNAQELBQAw
CzEJMAcGA1UEChMAMB4XDTE2MDUyMDExNTYyMFoXDTE3MDUyMDExNTYyMFowCzEJ
MAcGA1UEChMAMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAxgMXJUn2
H7TGjnBWmnVmZE8N4oVuFQ0CSekJA5QoiEJnR0l1NN0iWBlxtyeeWBamsYI23jXA
+HxZfRCwS7noYAWuvIO/qbG/HMdd093Y3QKsfs9NjYIldsXKjyO8rx8NjSI0yd/T
Uqj91ifke/ZRcoIIK9A/Dz0zX8+GWD8xM5SIheL6M2vLAFdVFeceb6tFRSqTq5fZ
zXq223XrNfVYhL5PC5wDB+QoaY6vb0K7r+fC1wDFuSfIihNnsXnJ2AJ3o+bLoc3o
oB9FcJY6BVtAxGfBiOL7VLHtQ29QwqSC8xuz14dcFYobhjRo0yc3ysoRaUeUwoln
G/PYSs/3yrvRQQIDAQABo3MwcTAOBgNVHQ8BAf8EBAMCAqQwGgYDVR0lBBMwEQYE
VR0lAAYJKwYBBAGCVwgGMAwGA1UdEwEB/wQCMAAwNQYDVR0RBC4wLIIJbG9jYWxo
b3N0gR9jaWFvLWRldmVsQGxpc3RzLmNsZWFybGludXgub3JnMA0GCSqGSIb3DQEB
CwUAA4IBAQAYFNWbmwfJsP+si02OwzGfE4zfsc+OKEMR7a0NBFrfSt1byfhiOdKz
eS64yqRW8a3DSgtsojeDa7uOJMxlYOWdpgf0lKJ91XJxvOY15D/9NPf+GpRSEp0v
GRjcVA9k1q0HvkskbF5F95qq6KdC15xwgWM6t91xwz1xBCU7X6XJ5YE1xblILUxr
Y4bXe+wY9Grj3rgcWySGVrZbgOFvD3i5pKOiFKMduPsnbvtLlIZ9bl73PqIjL2n0
X5O2B5+B2Jnh6XvqEoI4yFA3MgsDfPeR+L8hxWgSYQlT+uK37Koo8+EQX9RJgjtD
0wyigQNALie2QgUaVutamQ+AOBDJAc5/
-----END CERTIFICATE-----
-----BEGIN RSA PRIVATE KEY-----
MIIEpQIBAAKCAQEAxgMXJUn2H7TGjnBWmnVmZE8N4oVuFQ0CSekJA5QoiEJnR0l1
NN0iWBlxtyeeWBamsYI23jXA+HxZfRCwS7noYAWuvIO/qbG/HMdd093Y3QKsfs9N
jYIldsXKjyO8rx8NjSI0yd/TUqj91ifke/ZRcoIIK9A/Dz0zX8+GWD8xM5SIheL6
M2vLAFdVFeceb6tFRSqTq5fZzXq223XrNfVYhL5PC5wDB+QoaY6vb0K7r+fC1wDF
uSfIihNnsXnJ2AJ3o+bLoc3ooB9FcJY6BVtAxGfBiOL7VLHtQ29QwqSC8xuz14dc
FYobhjRo0yc3ysoRaUeUwolnG/PYSs/3yrvRQQIDAQABAoIBADijhala4JXtJaZ0
p7ECx8kFe9lBhV1sHS17BOMLLBTduaEAeBAo+Lvue0KCiJ51zDSWJI+nHI13NDm7
3lGq2bctqO+vV9F4UEwxEruZh4CgVSrorSw+/+xbYzdSZ5RH855dHHBqH45TXFg3
jPmQWXfBjgjKRl9biChtueXgHXi9EAj0I91o7Nlvknwb5z9w6Na6xXgRrkzmlZmo
/vB9vIlfl61tB7Vo8yenLZXvzckGXQGAmYNCxQT5UVCcI8I+uDmDIZCZ883PWRAf
Y/BsHMJIirwSYPNtoaudiKiUd3tE4UYpmBCKo9eLRRbIUmrBhtFNfzlKJOknIGKK
df+NSGkCgYEA/xsnsoevOkPfCLfJwP+aFHMEunw7GPo8BsKpZ6i1XmU7r9nsFNSp
arPS1PHQ+HW7Y4rsPDQtibOomSJbBPBxTbFAdDyRqRLrcB6PP2WT/WDaQ5rMBsvD
JMFLKDUEWkrWXfJKZH0JjCQGXD9JcQwL1MZdYx7sGxZvPjxLWvbPDNcCgYEAxrS3
/cT2aOHh3LfwsMbyL+yscQ7tOzRbJAH8HII71rz+CFxP91gCqSmkuu3J/FqJXvGn
aPAC5lI/WUe7l6t5GFr+Bu++XAOyFkbZqq7aqpHwWWekcoC0J9bZIJOQewkPtrQ6
SZYB0yutICH0NfliQRO2DBLd6mOU6eVKuuYR96cCgYEA4KOa610L8nS9u8GLicYK
eiNmcIjgyXfgz9surbckLsFaM9nkR9uUa/95kkZ2S7PwlRFKQSF5UB7CQka8e7b3
LwD8zt5fLdEZPZvLbHoYPTDCQnHXY8yeRIlpkzhMYu4von6u/5oThDJc33JjS4be
DDm5FoWuR5QT1Wvmt21KmfUCgYEAnozyxt2TlGqwcxKeTh/gfZcGnYvAhV4oXxnq
VcEhCB5zQK6P7Bfgv6QH2lQEuIRxWj3OS/A/EBKOk6mmKMZc8K8iRNylcvxL7HSK
GCQ/PJ0IP/5v/CFwHt3TEKhOP64iSLGUVMUKHbqbAKm3GB4uZSjaONYRFoZw+xTH
RUxxB78CgYEA27SGEWdLO77uKja4buFk+/ScXV8mlf54AR0u3GR7pXaHlZYOGv/w
lciys3+eEwnKCtkfL3teIRzENFbCqRFy/Wtbtl43J3u7OTEJaimDev2NZrqwoUUw
zeDHaZVad3qfU7X0z5eT+JZfi7yAcqEI3VEslBReWIIzHL5HR9rzyCU=
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
MIIDHjCCAgagAwIBAgIQH3ZiRo+kbCIiddp+7/65EzANBgkqhkiG9w0BAQsFADAL
MQkwBwYDVQQKEwAwHhcNMTYwNTE5MTI1MjE2WhcNMTcwNTE5MTI1MjE2WjALMQkw
BwYDVQQKEwAwggEiMA0GCSqGSIb3DQEBAQUAA4IBDwAwggEKAoIBAQC8BM8NNwd7
X4PQMgFFvToqO3HA3vfO2qeFlpYIzYIfbcXsEvSyFPmIDdobdpvNasuxKctc00UF
90rfHTR708hmI5kxm3h3jsCK2GE3Fxa0LH/Q5qoWq86ChKJNi0LLSY0KlmcDT2Ls
cbg+zO3//Ls5B/EJQW3ewRjnosKa61seWYXIJDXnzus6iTKxg0nvQhMau58jbNd6
73r4dPYDKEsAcqyTWkPunrB+PXQPbcXBkWmpFT0DQ64wujq670HwmIhn6OmtulXg
3wIcodp/+KbIHcvuzS8kxqfm9TIydxsHso3WuhRzoUa3EC3CoYZad3C/juJreRAG
8w7lhl+aaaiHAgMBAAGjfjB8MA4GA1UdDwEB/wQEAwICpDAlBgNVHSUEHjAcBgRV
HSUABgkrBgEEAYJXCAEGCSsGAQQBglcIBDAMBgNVHRMBAf8EAjAAMDUGA1UdEQQu
MCyCCWxvY2FsaG9zdIEfY2lhby1kZXZlbEBsaXN0cy5jbGVhcmxpbnV4Lm9yZzAN
BgkqhkiG9w0BAQsFAAOCAQEALmAU0GzKYHP9otk5KNI1ZGIXpneWPcxdNiynW4tx
pAHOfsy0KspcEXDVY97fQrVzgKqn9hU6U5sWy0k/2IKYn9m0pgSjAF5oLioKFpae
c2YpPPrB5BgUPpPEeA1ze0WrR638FAbsxFCzO6/xqXs6mkXW36wIk9a/35xK3zCY
iQdE93uCVYliHY4NdpZOc0KvKFHMscEhtIyTT3uEnsNp13lGp97u+JJR4qEh837P
2xhh0VETE0tNlB4/aZE6car+mtWXA9ZkI/46layvBO7hJwlIjgfkmhky2OSNfLTy
/XaSnPj9Q6OhtF68UmO+KxiTCvZezmua0x94qwsBfpGCAw==
-----END CERTIFICATE-----
-----BEGIN RSA PRIVATE KEY-----
MIIEowIBAAKCAQEAvATPDTcHe1+D0DIBRb06KjtxwN73ztqnhZaWCM2CH23F7BL0
shT5iA3aG3abzWrLsSnLXNNFBfdK3x00e9PIZiOZMZt4d47AithhNxcWtCx/0Oaq
FqvOgoSiTYtCy0mNCpZnA09i7HG4Pszt//y7OQfxCUFt3sEY56LCmutbHlmFyCQ1
587rOokysYNJ70ITGrufI2zXeu96+HT2AyhLAHKsk1pD7p6wfj10D23FwZFpqRU9
A0OuMLo6uu9B8JiIZ+jprbpV4N8CHKHaf/imyB3L7s0vJMan5vUyMncbB7KN1roU
c6FGtxAtwqGGWndwv47ia3kQBvMO5YZfmmmohwIDAQABAoIBABTyIDLjr4SyBlg6
SeQACavMxYZsEIVN3J3IQdynMFjZ/NOo5PO13HqouGSY2RCQVjLdahdkPetFOmUS
ttcYp9mhG57oKAqBr7eIFYRyoQffcTnPiKFP8IifyAkFe6J0Bi9ow/8dZ/LZVGJC
qDz9ZcobtWGHlXrcXi7n9fAWSideNt8O93D9j9L6w9WJ5f+VeRK5790wSwO8I/Hw
fkHfZBcq1Qm6hkpr1PJl+uaUdHihPVafJdi2yLzWnCcsUVGuhC3SBoNuNQzbpBWI
OCVbjTY42BfynQ/b7Mf73DFalNEXnB9JmX/9pIVoab4KqQOx746WBhz46GFhjcm2
sOiHRXkCgYEA1SZvseEmA89BQiJj38V7VTiTb4O9Kj6LCzUtQXcr9gK5vPc+K1/9
r9AQ3SZ7VPSvUa7IDv31xO1sWf47UwxGMNMPc/I0OZAv5JoREHK28yLQn1TjcEIY
dGZFSHJezw5PePbHlSlIaWuBWdF70TWeM8G1GY7A+qIkCQIjR7FOos0CgYEA4dED
AXxDZq5oRmJkDPJt5wGS3Tg+Kx0TNImVwAjuD/dX8U7qMxotmFuaDD+crEzpLXoF
fkBUWNlLY39KMupbvntH5hh8zUeCGHRsnWU99wvKaSDiK+q+3nn3rqkZ3mBjopId
mRObkpuTWIlzKUwoH2okYCmCAMQmL3biG3q4AKMCgYEAvQVW7AHj+mDjWEizFRBF
7S884BmNuVa5a3j+5x1NqN6F5GPFiCWaDT2Qlu23VYGfr+o1k8X3G2oJOk1QQreR
z158R7A0TA/nyOwv0cxJHZh0NbfL8hNLKH5BVpvGJAxmwbjnCQoRIxupHAO/r6nC
39caSM3lqN384tg4fS1ptMkCgYB54B/aLmIGSj03N6U+I53TXtMQGGndRQz3fwZW
pbsu3NUXBPt75zYSk+XZlH+Pstbq13+de0TKy0RYB/xY7InljY3pju/UrzJ01mlE
rb661h9BjCDliQXI91UJbHTsw2Mi+++DjcSaZOMqlsyTzmmdQuEtEVn75eGiHmy8
XghvNwKBgADM1JCPOKjV0KLieFUzGO6MSgddbZ7SEvlbHGmqJ2Z9uzv7GrwuZ1oN
Ez8n8SEhYVyZtVG+RpScJVHPnaon7yUHHdzVnDmiHkQ95g45+OrigzuQaUEZW8Nd
5Dizzh4+Ik4kUjV+JrFK9TDu4vUPYNfa7lDfKMvCdSraIudNzGEv
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
