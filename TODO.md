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


If I'd start fresh, I would:

* Make collection fully concurrent - top-level resources like Chassis, Systems, StorageServices, PowerEquipment, etc must all be scraped in parallel. It's children should also be scraped in parallel and so forth. gofish is fully thread-safe, but is limited to hardcoded 3 threads when querying collection (https://github.com/stmcginnis/gofish/blob/355941287fbc17e884fa00aa599f369e7d953ba7/common/collection.go#L125). Maybe submit a PR to make it tunable?
* There must be a per module configuration option to limit which paths should the exporter scrape. Some subtrees are VERY slow and not very useful to general public. But I believe it would be beneficial for community, if the new exporter was more flexible and extendable - if those features would not affect general public.
* I didn't run profiler, but it seems like creating a new collector from scratch every time an endpoint is called it superslow. Since we know "blueprints" for all types of Collectors at the very startup - from modules configuration - I strongly suggest initializing a CollectionFactory for each module and generating Collectors from them.
* TelemetryService might be something too look more closely at. I didn't like it, because for me it looked like it neither fits Prometheus, not OTEL approaches. I might be wrong though.
* OEM API is something we can benefit from, but it would require a community to maintain, as supporting all the changes from all the vendors might become a full time job. Atm I'd ignore it.


Note: exploring redfish API is not an easy task. OpenAPI docs generators just fail with OOMs as it's so huge. I'm using vscode "OpenAPI (Swagger) Editor" by 42Crunch - it at least gives me overview. You need to checkout gofish repo, run tools/gen_script.sh and look for tools/DSP*/openapi/openapi.yml.
