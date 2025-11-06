package webpage

// getMarketplaceTemplate returns the HTML header and styles for the marketplace
func getMarketplaceTemplate() string {
	return `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>REAPER Script Marketplace</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;
            margin: 0;
            padding: 20px;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            min-height: 100vh;
        }
        .container {
            max-width: 1200px;
            margin: 0 auto;
        }
        h1 {
            color: white;
            text-align: center;
            font-size: 2.5em;
            margin-bottom: 10px;
            text-shadow: 2px 2px 4px rgba(0,0,0,0.3);
        }
        .subtitle {
            color: rgba(255,255,255,0.9);
            text-align: center;
            margin-bottom: 30px;
            font-size: 1.1em;
        }
        .search-bar {
            margin-bottom: 30px;
            text-align: center;
        }
        .search-bar input {
            width: 100%;
            max-width: 600px;
            padding: 15px 20px;
            font-size: 16px;
            border: none;
            border-radius: 50px;
            box-shadow: 0 4px 6px rgba(0,0,0,0.1);
        }
        .scripts-grid {
            display: grid;
            grid-template-columns: repeat(auto-fill, minmax(350px, 1fr));
            gap: 20px;
        }
        .script-card {
            background: white;
            border-radius: 12px;
            padding: 20px;
            box-shadow: 0 4px 6px rgba(0,0,0,0.1);
            transition: transform 0.2s, box-shadow 0.2s;
        }
        .script-card:hover {
            transform: translateY(-5px);
            box-shadow: 0 8px 12px rgba(0,0,0,0.2);
        }
        .script-name {
            font-size: 1.3em;
            font-weight: bold;
            color: #333;
            margin-bottom: 10px;
        }
        .script-description {
            color: #666;
            margin-bottom: 15px;
            line-height: 1.5;
        }
        .script-meta {
            display: flex;
            gap: 10px;
            margin-bottom: 15px;
            flex-wrap: wrap;
        }
        .meta-badge {
            background: #f0f0f0;
            padding: 4px 10px;
            border-radius: 12px;
            font-size: 0.85em;
            color: #666;
        }
        .install-btn {
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
            border: none;
            padding: 12px 24px;
            border-radius: 8px;
            cursor: pointer;
            font-size: 1em;
            width: 100%;
            font-weight: 600;
            transition: opacity 0.2s;
        }
        .install-btn:hover {
            opacity: 0.9;
        }
        .install-btn:disabled {
            background: #ccc;
            cursor: not-allowed;
        }
        .installed-badge {
            background: #4caf50;
            color: white;
            padding: 8px 16px;
            border-radius: 8px;
            text-align: center;
            font-weight: 600;
        }
        .no-results {
            text-align: center;
            color: white;
            font-size: 1.2em;
            margin-top: 50px;
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>🎵 REAPER Script Marketplace</h1>
        <div class="subtitle">Browse and install ReaScripts for REAPER</div>

        <div class="search-bar">
            <input type="text" id="searchInput" placeholder="Search scripts..." onkeyup="filterScripts()">
        </div>

        <div class="scripts-grid" id="scriptsGrid">`
}

// getMarketplaceFooter returns the closing HTML and JavaScript for the marketplace
func getMarketplaceFooter() string {
	return `
        </div>
        <div class="no-results" id="noResults" style="display: none;">
            No scripts found matching your search.
        </div>
    </div>

    <script>
        function filterScripts() {
            const searchTerm = document.getElementById('searchInput').value.toLowerCase();
            const cards = document.querySelectorAll('.script-card');
            let visibleCount = 0;

            cards.forEach(card => {
                const name = card.getAttribute('data-name').toLowerCase();
                const description = card.getAttribute('data-description').toLowerCase();

                if (name.includes(searchTerm) || description.includes(searchTerm)) {
                    card.style.display = 'block';
                    visibleCount++;
                } else {
                    card.style.display = 'none';
                }
            });

            document.getElementById('noResults').style.display = visibleCount === 0 ? 'block' : 'none';
        }

        async function installScript(filename) {
            const btn = event.target;
            btn.disabled = true;
            btn.textContent = 'Installing...';

            try {
                // This would call back to ori-agent to execute the download_script operation
                // For now, just show success message
                alert('To install: Ask Ori to "download script ' + filename + '"');
                btn.textContent = 'Use Ori to Install';
            } catch (error) {
                alert('Error: ' + error.message);
                btn.disabled = false;
                btn.textContent = 'Install Script';
            }
        }
    </script>
</body>
</html>`
}
