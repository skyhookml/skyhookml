Vue.component('m-architecture', {
	data: function() {
		return {
			lines: [],
			inputs: [],
			outputs: [],
			addForms: null,
		};
	},
	props: ['arch'],
	created: function() {
		this.resetForm();

		var params = this.arch.Params;
		if(params.Arch) {
			this.lines = params.Arch;
		}
		if(params.Inputs) {
			this.inputs = params.Inputs;
		}
		if(params.Outputs) {
			this.outputs = params.Outputs;
		}
	},
	methods: {
		save: function() {
			let params = {
				Inputs: this.inputs,
				Arch: this.lines,
				Outputs: this.outputs,
			};
			myCall('POST', '/keras/archs/'+this.arch.ID, JSON.stringify({
				Params: params,
			}));
		},
		resetForm: function() {
			this.addForms = {
				lineName: '',
				lineCode: '',
				inputName: '',
				inputCode: '',
				output: '',
			};
		},
		addLine: function() {
			this.lines.push([this.addForms.lineName, this.addForms.lineCode]);
			this.resetForm();
		},
		removeLine: function(i) {
			this.lines.splice(i, 1);
		},
		addInput: function() {
			this.inputs.push([this.addForms.inputName, this.addForms.inputCode]);
			this.resetForm();
		},
		removeInput: function(i) {
			this.inputs.splice(i, 1);
		},
		addOutput: function() {
			this.outputs.push(this.addForms.output);
			this.resetForm();
		},
		removeOutput: function(i) {
			this.outputs.splice(i, 1);
		},
	},
	template: `
<div class="small-container m-2">
	<div class="form-group row">
		<label class="col-sm-2 col-form-label">Architecture</label>
		<div class="col-sm-10">
			<table class="table">
				<tbody>
					<tr v-for="(line, i) in lines">
						<td>{{ line[0] }}</td>
						<td>{{ line[1] }}</td>
						<td>
							<button type="button" class="btn btn-danger" v-on:click="removeLine(i)">Remove</button>
						</td>
					</tr>
					<tr>
						<td>
							<input v-model="addForms.lineName" type="text" class="form-control">
						</td>
						<td>
							<input v-model="addForms.lineCode" type="text" class="form-control">
						</td>
						<td>
							<button type="button" class="btn btn-primary" v-on:click="addLine">Add</button>
						</td>
					</tr>
				</tbody>
			</table>
		</div>
	</div>
	<div class="form-group row">
		<label class="col-sm-2 col-form-label">Inputs</label>
		<div class="col-sm-10">
			<table class="table">
				<tbody>
					<tr v-for="(inp, i) in inputs">
						<td>{{ inp[0] }}</td>
						<td>{{ inp[1] }}</td>
						<td>
							<button type="button" class="btn btn-danger" v-on:click="removeInput(i)">Remove</button>
						</td>
					</tr>
					<tr>
						<td>
							<input v-model="addForms.inputName" type="text" class="form-control">
						</td>
						<td>
							<input v-model="addForms.inputCode" type="text" class="form-control">
						</td>
						<td>
							<button type="button" class="btn btn-primary" v-on:click="addInput">Add</button>
						</td>
					</tr>
				</tbody>
			</table>
		</div>
	</div>
	<div class="form-group row">
		<label class="col-sm-2 col-form-label">Outputs</label>
		<div class="col-sm-10">
			<table class="table">
				<tbody>
					<tr v-for="(s, i) in outputs">
						<td>{{ s }}</td>
						<td>
							<button type="button" class="btn btn-danger" v-on:click="removeOutput(i)">Remove</button>
						</td>
					</tr>
					<tr>
						<td>
							<input v-model="addForms.output" type="text" class="form-control">
						</td>
						<td>
							<button type="button" class="btn btn-primary" v-on:click="addOutput">Add</button>
						</td>
					</tr>
				</tbody>
			</table>
		</div>
	</div>
	<button v-on:click="save" type="button" class="btn btn-primary">Save</button>
</div>
	`,
});
