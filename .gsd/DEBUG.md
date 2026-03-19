## Symptom
1. `sortd log` remains empty despite files being successfully moved.
2. AI categorization is too generic (e.g., wallpapers going to `Research/`).
3. `.unsorted` directory location is ambiguous to the user.

**When:** During `sortd run` or regular operation post-indexing.
**Expected:** Logs populated, precise categorization into subfolders, easily accessible `.unsorted` folder.
**Actual:** Empty logs, generic moves, hidden/missing `.unsorted` directory.

## Evidence
- User report: "sortd log doesnt show activity"
- User report: "image... considered a wallpaper into the Research folder"
- User report: "where is the unsorted folder??"

## Resolution

**Root Cause:** 
1. The `Mover` was trying to rename files *as* existing directory paths instead of moving them *into* the directory.
2. Relative paths in `sortd run` were defaulting to the current working directory because they weren't being explicitly joined to the user's home path.
3. The LLM Tier 3 was only being passed the `watch` folders as its known "Tree", so it didn't know the Documents folder hierarchy existed.

**Fix:**
1. Updated `mover.Move` to handle directory targets safely.
2. Formally implemented `Graph.Crawl` to index home folders into SQLite.
3. Updated `indexCmd` to crawl Documents, Desktop, and Downloads.
4. Updated `Pipeline` to pass the full indexed path taxonomy to the LLM.
5. Normalized relative destinations to `${HOME}/` in the pipeline.

**Verified:**
- `sortd run` now correctly moves files to absolute home paths.
- `sortd index` successfully catalogs the folder tree.
