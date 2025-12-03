TODOs (in no particular order):

* Collection of different API branches is done sequentially. The code could be significantly sped up by implementing full concurrency. Request limiting can be handled at the `APIClient` level.
* Allow configuration of which API sections to pull and which to ignore — another way to speed things up.
* Manager collector fails for some reason.
* Many top-level resources are not implemented.
* Many middle- and leaf-level resources are not implemented either.
* No OEM API resources support.
* The code must either fail gracefully and serve best-effort metrics, or fail fast and serve none. Currently it may get stuck (on a timed-out request to the Redfish API? Not sure what the root cause is, it's rare) and never return, which is a disaster.
* Metrics and labels do not meet best-practice criteria.
* The code is not fully up to date with the latest version of the API. Check the `gofish` release notes — I updated the lib from version 0.14.0 to 0.20.0, probably missed a lot, and only fixed errors if they prevented compilation.
