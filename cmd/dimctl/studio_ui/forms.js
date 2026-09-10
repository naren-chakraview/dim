class FormGenerator {
	constructor(schemaPath = "/schemas/route.json") {
		this.schema = null;
		this.load(schemaPath);
	}

	async load(schemaPath) {
		try {
			const response = await fetch(schemaPath);
			this.schema = await response.json();
		} catch (error) {
			console.warn("Could not load schema:", error);
			this.schema = {};
		}
	}

	generateForm(routeName, routeData) {
		const form = document.createElement("form");
		form.className = "route-form";

		// Simple example: render fields for route-level config
		const fieldsToShow = ["auth", "lineage"];
		for (const field of fieldsToShow) {
			const value = routeData[field] || "";
			const input = document.createElement("input");
			input.type = "text";
			input.placeholder = field;
			input.value = value;
			input.dataset.field = field;

			const label = document.createElement("label");
			label.textContent = field;
			label.appendChild(input);

			form.appendChild(label);
		}

		// Save button
		const saveBtn = document.createElement("button");
		saveBtn.textContent = "Save";
		saveBtn.type = "button";
		saveBtn.addEventListener("click", () => this.onSave(routeName, form));
		form.appendChild(saveBtn);

		return form;
	}

	async onSave(routeName, form) {
		const editData = {};
		form.querySelectorAll("input").forEach((input) => {
			editData[input.dataset.field] = input.value;
		});

		try {
			const response = await fetch(`/api/routes?route=${routeName}`, {
				method: "POST",
				headers: { "Content-Type": "application/json" },
				body: JSON.stringify(editData),
			});

			const result = await response.json();
			if (result.status === "ok") {
				alert("Route saved!");
			} else {
				alert(`Save failed: ${result.error}`);
			}
		} catch (error) {
			alert(`Save error: ${error.message}`);
		}
	}
}
