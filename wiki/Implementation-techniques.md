[[_TOC_]]

---

# Implementation techniques

The page describes how we can use external APIs implement our solution.

---

## Retrieving charts from Power BI

### API way

The [Reports - Export To File](https://docs.microsoft.com/en-us/rest/api/power-bi/reports/exporttofile/) endpoint allows us to create a rendition of a report.
Following formats are suitable for us (as Slack will show a preview for such a message): PNG or PDF.

Concern 1: As we cannot obtain a rendition of a single visual, message preview for a report w/ too many visuals won't be readable as-is -- opening it in full screen will be required.

Concern 2: How should we handle multi-page reports?
* We can show a message per page.
  * What if there are too many pages in report?
    It would be good to offer our user a choice on what page they want to see in Slack (see [Reports - Get Pages](https://docs.microsoft.com/en-us/rest/api/power-bi/reports/getpages/)).
* Multi-page PDFs can be viewed directly within Slack -- there's no special need in having dedicated messages for each page.

Concern 3: Exporting to PDF isn't instantaneous -- it takes a minute or so -- that may hurt our UX. For other limitations see [Export reports to PDF](https://docs.microsoft.com/en-us/power-bi/consumer/end-user-pdf/).

Concern 4: How to offer a choice on what report we should display? [Reports - Get Reports](https://docs.microsoft.com/en-us/rest/api/power-bi/reports/getreports/) endpoint allows us to obtain a list of reports.

Concern 5: It seems that MS offers us no way to apply filters before export. [This feature](https://ideas.powerbi.com/forums/265200-power-bi-ideas/suggestions/40060261-add-filter-options-to-the-export-file-api%20) is planned to be in the future release.

Concern 6: Using the file export API requires for the report & its dataset to be on a dedicated capacity (see [Export Power BI reports API](https://docs.microsoft.com/en-us/power-bi/developer/embedded/export-to#limitations)), i.e. a customer must have an additional costly Power BI Premium or Embedded subscription, Free or even Pro ones won't work. This API limitation may reduce our potential userbase & introduce an impediment for us during development & testing as we don't have any of these subscriptions.

### Embedding way

Power BI also has another way of sharing reports -- embedding ([Embed a report in a secure portal or website](https://docs.microsoft.com/en-us/power-bi/collaborate-share/service-embed-secure/).
Embedded reports can be filtered using query string parameters, pages can be selected too.
They require user authentication upon viewing -- this method is designed for publishing to internal websites.

Publishing to web ([Publish to web](https://docs.microsoft.com/en-us/power-bi/collaborate-share/service-publish-to-web/)) doesn't require authentication, but it doesn't allow filtering.

Also there seems to be no API to create a rendition for such embedded reports -- we'll have to spin up a browser, navigate to report's URL, & make a screenshot.

However, we shouldn't ignore embedded reports as a use case -- we can add a feature to our bot to unfurl (see [Unfurling links in messages](https://api.slack.com/reference/messaging/link-unfurling/)) such URLs to add neat previews for messages containing them.

Power BI Embedded is another solution oriented to software developers.
We have full programmatic access to filters & other report features.
This method still isn't suitable for us -- it isn't designed to work in case when reports we handle aren't our own but user's.
Also it's a separate service that adds additional cost.
See demo here: [Microsoft Power BI Embedded Playground](https://microsoft.github.io/PowerBI-JavaScript/demo/v2-demo/index.html).

### Export to PDF (feature on PowerBI website)

This is a free feature to export a report to PDF which is available on the webstire only. But there is a workaround how to use it.
1. Execute http-get request (Rest API endpoint):\
https://api.powerbi.com/v1.0/myorg/groups/{group_id}/reports/{report_id}
2. From the response get a value of the parameter "@odata.context"
3. Get a hostname of the value from step 2
4. Execute http-post request (undocumented endpoint):\
https://{hostname_from_step_3}/export/reports/{report_id}/asyncexports
5. From the response get a value of the parameter "id"
6. Execute http-get request (undocumented endpoint):\
https://{hostname_from_step_3}/export/reports/{report_id}/asyncexports/{id_from_step_5}/status
7. From the response check a value of the parameter "status".
8. If value of the step 7 equals "3" then the file is ready
9. Execute http-get request to download the file (undocumented endpoint):\
https://{hostname_from_step_3}/export/reports/{report_id}/asyncexports/{id_from_step_5}/file 

### Conclusion

Embed report to html page and get a screenshot
