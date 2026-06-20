---
SourceURL: https://owasp.org/API-Security/editions/2023/en/0x11-t10/
FetchedAt: 2026-06-18T17:42:21-04:00
---

## OWASP Top 10 API Security Risks – 2023

| Risk | Description |
| --- | --- |
| [API1:2023 - Broken Object Level 
Authorization](https://owasp.org/API-Security/editions/2023/en/0xa1-broken-object-level-authorizatio
n/) | APIs tend to expose endpoints that handle object identifiers, creating a wide attack surface 
of Object Level Access Control issues. Object level authorization checks should be considered in 
every function that accesses a data source using an ID from the user. |
| [API2:2023 - Broken 
Authentication](https://owasp.org/API-Security/editions/2023/en/0xa2-broken-authentication/) | 
Authentication mechanisms are often implemented incorrectly, allowing attackers to compromise 
authentication tokens or to exploit implementation flaws to assume other user's identities 
temporarily or permanently. Compromising a system's ability to identify the client/user, 
compromises API security overall. |
| [API3:2023 - Broken Object Property Level 
Authorization](https://owasp.org/API-Security/editions/2023/en/0xa3-broken-object-property-level-aut
horization/) | This category combines [API3:2019 Excessive Data 
Exposure](https://owasp.org/API-Security/editions/2019/en/0xa3-excessive-data-exposure/) and 
[API6:2019 - Mass 
Assignment](https://owasp.org/API-Security/editions/2019/en/0xa6-mass-assignment/), focusing on the 
root cause: the lack of or improper authorization validation at the object property level. This 
leads to information exposure or manipulation by unauthorized parties. |
| [API4:2023 - Unrestricted Resource 
Consumption](https://owasp.org/API-Security/editions/2023/en/0xa4-unrestricted-resource-consumption/
) | Satisfying API requests requires resources such as network bandwidth, CPU, memory, and storage. 
Other resources such as emails/SMS/phone calls or biometrics validation are made available by 
service providers via API integrations, and paid for per request. Successful attacks can lead to 
Denial of Service or an increase of operational costs. |
| [API5:2023 - Broken Function Level 
Authorization](https://owasp.org/API-Security/editions/2023/en/0xa5-broken-function-level-authorizat
ion/) | Complex access control policies with different hierarchies, groups, and roles, and an 
unclear separation between administrative and regular functions, tend to lead to authorization 
flaws. By exploiting these issues, attackers can gain access to other users’ resources and/or 
administrative functions. |
| [API6:2023 - Unrestricted Access to Sensitive Business 
Flows](https://owasp.org/API-Security/editions/2023/en/0xa6-unrestricted-access-to-sensitive-busines
s-flows/) | APIs vulnerable to this risk expose a business flow - such as buying a ticket, or 
posting a comment - without compensating for how the functionality could harm the business if used 
excessively in an automated manner. This doesn't necessarily come from implementation bugs. |
| [API7:2023 - Server Side Request 
Forgery](https://owasp.org/API-Security/editions/2023/en/0xa7-server-side-request-forgery/) | 
Server-Side Request Forgery (SSRF) flaws can occur when an API is fetching a remote resource 
without validating the user-supplied URI. This enables an attacker to coerce the application to 
send a crafted request to an unexpected destination, even when protected by a firewall or a VPN. |
| [API8:2023 - Security 
Misconfiguration](https://owasp.org/API-Security/editions/2023/en/0xa8-security-misconfiguration/) 
| APIs and the systems supporting them typically contain complex configurations, meant to make the 
APIs more customizable. Software and DevOps engineers can miss these configurations, or don't 
follow security best practices when it comes to configuration, opening the door for different types 
of attacks. |
| [API9:2023 - Improper Inventory 
Management](https://owasp.org/API-Security/editions/2023/en/0xa9-improper-inventory-management/) | 
APIs tend to expose more endpoints than traditional web applications, making proper and updated 
documentation highly important. A proper inventory of hosts and deployed API versions also are 
important to mitigate issues such as deprecated API versions and exposed debug endpoints. |
| [API10:2023 - Unsafe Consumption of 
APIs](https://owasp.org/API-Security/editions/2023/en/0xaa-unsafe-consumption-of-apis/) | 
Developers tend to trust data received from third-party APIs more than user input, and so tend to 
adopt weaker security standards. In order to compromise APIs, attackers go after integrated 
third-party services instead of trying to compromise the target API directly. |
