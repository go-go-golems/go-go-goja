fetch('/api/config')
  .then((response) => response.json())
  .then((data) => {
    document.getElementById('status').textContent = 'embedded asset bundle: ' + data.config.name
  })
  .catch((err) => {
    document.getElementById('status').textContent = String(err)
  })
