let canvas;
let formGenerator;

document.addEventListener("DOMContentLoaded", async () => {
	canvas = new RouteCanvas(document.getElementById("canvas"));
	formGenerator = new FormGenerator();

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
	inspector.innerHTML = "";

	// Show form for selected node
	if (nodeId.includes("filter") || nodeId.includes("translate")) {
		const editor = new JSONataEditor(inspector);
		editor.setValue("body.field");
	} else {
		const form = formGenerator.generateForm(nodeId, {});
		inspector.appendChild(form);
	}
});
