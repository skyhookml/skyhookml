import utils from './utils.js';

export default {
	data: function() {
		return {
			name: '',
			outputs: [],
			addOutputForm: null,
			op: null,
			categories: [
				{
					ID: "basic",
					Name: "Basic",
					Ops: [
						{
							ID: "filter",
							Name: "Filter",
							Description: "Filter",
							Inputs: [{Name: "inputs", Variable: true}],
							Outputs: [{Name: "output", DataType: "int"}],
						},
						{
							ID: "detection_filter",
							Name: "Detection Filter",
							Description: "Detection Filter",
							Inputs: [{Name: "detections", DataTypes: ["detection"]}],
							Outputs: [{Name: "detections", DataType: "detection"}],
						},
						{
							ID: "simple_tracker",
							Name: "Simple Tracker",
							Description: "Simple Tracker",
							Inputs: [{Name: "detections", DataTypes: ["detection"]}],
							Outputs: [{Name: "tracks", DataType: "detection"}],
						},
						{
							ID: "reid_tracker",
							Name: "Reid Tracker",
							Description: "Tracker using Re-identification Model",
							Inputs: [
								{Name: "model", DataTypes: ["string"]},
								{Name: "video", DataTypes: ["video"]},
								{Name: "detections", DataTypes: ["detection"]},
							],
							Outputs: [{Name: "tracks", DataType: "detection"}],
						},
						{
							ID: "resample",
							Name: "Resample",
							Description: "Resample sequence data at a different rate",
							Inputs: [{Name: "inputs", Variable: true}],
							Outputs: [],
						},
					],
				},
				{
					ID: "model",
					Name: "Model",
					Ops: [
						{
							ID: "pytorch_train",
							Name: "Pytorch (train)",
							Description: "Pytorch (train)",
							Inputs: [
								{Name: "inputs", Variable: true},
								{Name: "models", DataTypes: ["string"], Variable: true},
							],
							Outputs: [{Name: "model", DataType: "string"}],
						},
						{
							ID: "pytorch_infer",
							Name: "Pytorch (infer)",
							Description: "Pytorch (infer)",
							Inputs: [
								{Name: "model", DataTypes: ["string"]},
								{Name: "inputs", Variable: true},
							],
							Outputs: [],
						},
						{
							ID: "yolov3_train",
							Name: "Yolov3 (train)",
							Description: "Yolov3 (train)",
							Inputs: [
								{Name: "images", DataTypes: ["video", "image"]},
								{Name: "detections", DataTypes: ["detection"]},
							],
							Outputs: [{Name: "model", DataType: "string"}],
						},
						{
							ID: "yolov3_infer",
							Name: "Yolov3 (infer)",
							Description: "Yolov3 (infer)",
							Inputs: [
								{Name: "model", DataTypes: ["string"]},
								{Name: "images", DataTypes: ["video", "image"]},
							],
							Outputs: [{Name: "detections", DataType: "detection"}],
						},
						{
							ID: "unsupervised_reid",
							Name: "Unsupervised Re-identification",
							Description: "Self-Supervised Re-identification Model",
							Inputs: [
								{Name: "video", DataTypes: ["video"]},
								{Name: "detections", DataTypes: ["detection"]},
							],
							Outputs: [{Name: "model", DataType: "string"}],
						},
					],
				},
				{
					ID: "video",
					Name: "Video Manipulation",
					Ops: [
						{
							ID: "video_sample",
							Name: "Sample video",
							Description: "Sample images or segments from video",
							Inputs: [
								{Name: "video", DataTypes: ["video"]},
								{Name: "others", Variable: true},
							],
							// could also be video, but we'll update it in the node editor
							Outputs: [{Name: "samples", DataType: "image"}],
						},
						{
							ID: "render",
							Name: "Render video",
							Description: "Render video from various input data types",
							Inputs: [{Name: "inputs", Variable: true}],
							Outputs: [{Name: "output", DataType: "video"}],
						},
						{
							ID: "cropresize",
							Name: "Crop/Resize Video",
							Description: "Crop video followed by optional resize",
							Inputs: [{Name: "input", DataTypes: ["video"]}],
							Outputs: [{Name: "output", DataType: "video"}],
						},
					],
				},
				{
					ID: "code",
					Name: "Code",
					Ops: [
						{
							ID: "python",
							Name: "Python",
							Description: "Express a Python function for the system to execute",
							Inputs: [{Name: "inputs", Variable: true}],
						},
					],
				},
			],
		};
	},
	created: function() {
		this.resetForm();
	},
	mounted: function() {
		$(this.$refs.modal).modal('show');
	},
	methods: {
		createNode: function() {
			var params = {
				Name: this.name,
				Op: this.op.ID,
				Params: '',
				Parents: null,
				Inputs: this.op.Inputs,
				Outputs: this.outputs,
				Workspace: this.$route.params.ws,
			};
			utils.request(this, 'POST', '/exec-nodes', JSON.stringify(params), () => {
				$(this.$refs.modal).modal('hide');
				this.$emit('closed');
			});
		},
		selectOp: function(op) {
			this.op = op;
			if(op.Outputs) {
				this.outputs = op.Outputs;
			} else {
				this.outputs = [];
			}
		},
		resetForm: function() {
			this.addOutputForm = {
				name: '',
				dataType: '',
			};
		},
		addOutput: function() {
			this.outputs.push({
				Name: this.addOutputForm.name,
				DataType: this.addOutputForm.dataType,
			});
			this.resetForm();
		},
		removeOutput: function(i) {
			this.outputs.splice(i, 1);
		},
	},
	template: `
<div class="modal" tabindex="-1" role="dialog" ref="modal">
	<div class="modal-dialog modal-xl" role="document">
		<div class="modal-content">
			<div class="modal-body">
				<form v-on:submit.prevent="createNode">
					<div class="form-group row">
						<label class="col-sm-2 col-form-label">Name</label>
						<div class="col-sm-10">
							<input v-model="name" class="form-control" type="text" />
						</div>
					</div>
					<div class="form-group row">
						<label class="col-sm-2 col-form-label">Op</label>
						<div class="col-sm-10">
							<ul class="nav nav-tabs">
								<li v-for="category in categories" class="nav-item">
									<a
										class="nav-link"
										data-toggle="tab"
										:href="'#add-node-cat-' + category.ID"
										role="tab"
										>
										{{ category.Name }}
									</a>
								</li>
							</ul>
							<div class="tab-content">
								<div v-for="category in categories" class="tab-pane" :id="'add-node-cat-' + category.ID">
									<table class="table table-row-select">
										<thead>
											<tr>
												<th>Name</th>
												<th>Description</th>
											</tr>
										</thead>
										<tbody>
											<tr
												v-for="x in category.Ops"
												:class="{selected: op != null && op.ID == x.ID}"
												v-on:click="selectOp(x)"
												>
												<td>{{ x.Name }}</td>
												<td>{{ x.Description }}</td>
											</tr>
										</tbody>
									</table>
								</div>
							</div>
						</div>
					</div>
					<div class="form-group row">
						<label class="col-sm-2 col-form-label">Outputs</label>
						<div class="col-sm-10">
							<table v-if="op != null" class="table">
								<thead>
									<tr>
										<th>Name</th>
										<th>Type</th>
										<th v-if="!op.Outputs"></th>
									</tr>
								</thead>
								<tbody>
									<tr v-for="(output, i) in outputs">
										<td>{{ output.Name }}</td>
										<td>{{ output.DataType }}</td>
										<td v-if="!op.Outputs">
											<button type="button" class="btn btn-danger" v-on:click="removeOutput(i)">Remove</button>
										</td>
									</tr>
									<tr v-if="!op.Outputs">
										<td>
											<input type="text" class="form-control" v-model="addOutputForm.name" />
										</td>
										<td>
											<select v-model="addOutputForm.dataType" class="form-control">
												<option v-for="(dt, name) in $globals.dataTypes" :value="dt">{{ name }}</option>
											</select>
										</td>
										<td>
											<button type="button" class="btn btn-primary" v-on:click="addOutput">Add</button>
										</td>
									</tr>
								</tbody>
							</table>
						</div>
					</div>
					<div class="form-group row">
						<div class="col-sm-10">
							<button type="submit" class="btn btn-primary">Create Node</button>
						</div>
					</div>
				</form>
			</div>
		</div>
	</div>
</div>
	`,
};
