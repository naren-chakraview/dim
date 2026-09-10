class JSONataEditor {
	constructor(containerElement) {
		this.container = containerElement;
		this.editor = null;
		this.init();
	}

	init() {
		this.editor = document.createElement("textarea");
		this.editor.className = "jsonata-editor";
		this.editor.placeholder = "JSONata expression (e.g., body.id)";
		this.container.appendChild(this.editor);

		// Basic syntax highlighting simulation
		this.editor.addEventListener("input", () => this.highlight());
	}

	setValue(expr) {
		this.editor.value = expr;
		this.highlight();
	}

	getValue() {
		return this.editor.value;
	}

	highlight() {
		// TODO: Add syntax highlighting (simple regex-based for now)
	}
}
