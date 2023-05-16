[[_TOC_]]

---

# Known issues

When you encounter an issue w/ tooling, libraries, & APIs, add a workaround here to share w/ the team.

---

## Report rendering performance

As per results of load testing conducted on 2020-11-19 (on a `t2.micro` instance for staging environment), we have following issues:

| Load type | Load rate (#requests per period) | Peak RAM usage (GiB) | Average processing time (seconds) | Conclusion |
| --- | --- | --- | --- | --- |
| One-time | 50 per second | 🛑 1.8 | 🛑 N/A | 🛑 Not capable to perform the task |
| One-time | 10 per second | ⚠ 1.4 | 80 - 160 | ⚠ RAM limit exceeded |
| Continuous | 8 per minute | 0.7 | 25 - 80 | Processing time is OK, but there's not enough capacity for production usage |
| Continuous | 1 - 4 per minute | 0.5 | 20 | 〃 |
