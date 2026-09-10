class RouteCanvas {
	constructor(containerElement) {
		this.container = containerElement;
		this.svg = null;
		this.routes = {};
		this.selectedNode = null;
		this.init();
	}

	init() {
		// Create SVG element
		this.svg = document.createElementNS("http://www.w3.org/2000/svg", "svg");
		this.svg.setAttribute("width", "100%");
		this.svg.setAttribute("height", "100%");
		this.container.appendChild(this.svg);

		// Add zoom/pan support (basic)
		this.svg.addEventListener("wheel", (e) => this.onZoom(e));
	}

	render(routes) {
		this.routes = routes;
		this.svg.innerHTML = ""; // Clear

		// Group for edges
		const edgeGroup = document.createElementNS("http://www.w3.org/2000/svg", "g");
		edgeGroup.setAttribute("class", "edges");
		this.svg.appendChild(edgeGroup);

		// Group for nodes
		const nodeGroup = document.createElementNS("http://www.w3.org/2000/svg", "g");
		nodeGroup.setAttribute("class", "nodes");
		this.svg.appendChild(nodeGroup);

		// Layout nodes (simple grid for now)
		let x = 50, y = 50;
		const nodePositions = {};

		for (const [routeName, routeData] of Object.entries(routes)) {
			const route = routeData.data.routes?.[routeName] || {};
			const steps = route.steps || [];

			// Draw source node
			const sourceNode = this.createNode(routeName + "-source", route.from || "unknown", x, y, "#4CAF50");
			nodeGroup.appendChild(sourceNode);
			nodePositions[route.from] = { x, y };

			y += 100;

			// Draw step nodes
			for (const step of steps) {
				const stepType = Object.keys(step)[0];
				const stepNode = this.createNode(stepType, stepType, x, y, "#2196F3");
				nodeGroup.appendChild(stepNode);
				y += 100;
			}

			// Draw sink nodes
			const sinks = routeData.data.sinks || {};
			for (const [sinkName] of Object.entries(sinks)) {
				const sinkNode = this.createNode(sinkName, sinkName, x + 300, y, "#FF9800");
				nodeGroup.appendChild(sinkNode);
				y += 100;
			}

			x += 500;
			y = 50;
		}
	}

	createNode(id, label, x, y, color) {
		const g = document.createElementNS("http://www.w3.org/2000/svg", "g");
		g.setAttribute("class", "node");
		g.setAttribute("id", id);
		g.setAttribute("transform", `translate(${x},${y})`);

		// Rectangle
		const rect = document.createElementNS("http://www.w3.org/2000/svg", "rect");
		rect.setAttribute("width", "120");
		rect.setAttribute("height", "60");
		rect.setAttribute("rx", "4");
		rect.setAttribute("fill", color);
		rect.setAttribute("stroke", "#333");
		rect.setAttribute("stroke-width", "2");
		g.appendChild(rect);

		// Label
		const text = document.createElementNS("http://www.w3.org/2000/svg", "text");
		text.setAttribute("x", "60");
		text.setAttribute("y", "35");
		text.setAttribute("text-anchor", "middle");
		text.setAttribute("fill", "#fff");
		text.setAttribute("font-size", "12");
		text.textContent = label;
		g.appendChild(text);

		// Click handler
		g.addEventListener("click", () => this.onNodeSelected(id));

		return g;
	}

	onNodeSelected(nodeId) {
		if (this.selectedNode) {
			this.selectedNode.setAttribute("opacity", "1");
		}
		const node = document.getElementById(nodeId);
		node.setAttribute("opacity", "0.7");
		this.selectedNode = node;

		// Emit event for inspector panel
		document.dispatchEvent(new CustomEvent("nodeSelected", { detail: { nodeId } }));
	}

	onZoom(e) {
		e.preventDefault();
		// Simple zoom (not fully implemented)
	}
}
