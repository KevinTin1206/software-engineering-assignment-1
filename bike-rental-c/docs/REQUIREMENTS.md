# Requirements trace

| Requirement | Evidence in project |
|---|---|
| Go, memory, HTML/JS | go.mod, main.go, internal/store, web |
| a: Dashboard | Dashboard counters, open workflow list, quick actions |
| c: Bicycle rental | Item/person CRUD, reserve, checkout, return, cancellation |
| Rental beginning/duration | Inclusive start/end dates, validated and tested |
| Identity information and image | Person fields and validated in-memory JPG/PNG endpoint; preview/replace/delete UI |
| b: Flexible prices | Category rate editor, arbitrary duration tiers, quote snapshot |
| c: Separate bicycle project | This independent project with own Go module and Git; City/Trekking/E-Bike inventory |
| Reuse reasonable design | Same tested domain/store/handler/assets; domain labels, categories and seed differ |
| README | Startup, demo, business rules, API, scope, Chinese quick-start |
| Automated tests | Domain/store/httptest/application smoke tests |
| Git | Independent local project repository and incremental commits |
| Packaging | Independent source ZIP; no database, npm or sibling module needed |

Source: supplied assignment screenshots and teacher's Template CLAUDE.rtf.
Unclear handwritten/lecture details are listed in DESIGN.md and README.md;
no claim is made that these reproduce the unseen classroom application exactly.
