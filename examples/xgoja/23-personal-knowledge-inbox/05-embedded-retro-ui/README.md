# Step 05: embedded retro UI

This step copies Step 04 and adds embedded browser assets. The API and CLI client remain public and unauthenticated; the new concept is that the generated binary can embed static HTML, CSS, and browser JavaScript, then serve those files from the xgoja HTTP server.

The UI style is intentionally constrained:

- retro monochrome Macintosh-inspired layout,
- no menu bar or window chrome,
- no Chicago font,
- modern Swiss/system typography,
- mostly black-on-warm-paper text,
- muted accent colors only for small foreground details,
- thin rules instead of boxes everywhere.

## Files

```text
assets/public/index.html
assets/public/styles.css
assets/public/app.js
verbs/server.js
verbs/client.js
verbs/lib/inbox_store.js
verbs/lib/api_client.js
```

The server mounts the embedded assets with:

```js
const assets = require("fs:assets")
app.staticFromAssetsModule("/static", assets, "/app/public")
```

and serves `/` from the embedded `index.html`.

## Run

```bash
make smoke
```

Manual run after `make build`:

```bash
./dist/personal-knowledge-inbox-embedded-ui \
  serve inbox server \
  --http-listen 127.0.0.1:18792 \
  --db /tmp/personal-inbox-ui.sqlite
```

Then open:

- <http://127.0.0.1:18792/>
- <http://127.0.0.1:18792/static/styles.css>
- <http://127.0.0.1:18792/static/app.js>

Later steps will protect the API and UI with session and programmatic auth.
