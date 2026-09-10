let canvas;

document.addEventListener("DOMContentLoaded", async () => {
	canvas = new RouteCanvas(document.getElementById("canvas"));

	// Load routes from API
	try {
		const response = await fetch("/api/routes");
		const json = await response.json();

		if (json.routes) {
			canvas.render(json.routes);
			document.getElementById("status").textContent = `Loaded ${Object.keys(json.routes).length} route(s)`;
		}
	} catch (error) {
		document.getElementById("status").textContent = `Error loading routes: ${error.message}`;
	}
});

document.addEventListener("nodeSelected", (e) => {
	const { nodeId } = e.detail;
	const inspector = document.getElementById("properties");
	inspector.innerHTML = `<p>Selected: <strong>${nodeId}</strong></p>`;
});
