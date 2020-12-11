import utils from './utils.js';

export default {
	data: function() {
		return {
			name: '',
			op: null,
			categories: [
				{
					ID: "models",
					Name: "Models",
					Ops: [
						{
							ID: "keras",
							Name: "Keras",
							Description: "Keras",
						},
						{
							ID: "pytorch",
							Name: "Pytorch",
							Description: "Pytorch",
						},
						{
							ID: "yolov3",
							Name: "YOLOv3",
							Description: "YOLOv3 Object Detector (darknet implementation)",
						},
					],
				},
			],
		};
	},
	mounted: function() {
		$(this.$refs.modal).modal('show');
	},
	methods: {
		createNode: function() {
			var params = {
				name: this.name,
				op: this.op,
				ws: this.$route.params.ws,
			};
			utils.request(this, 'POST', '/train-nodes', params, () => {
				$(this.$refs.modal).modal('hide');
				this.$emit('closed');
			});
		},
		selectOp: function(op) {
			this.op = op;
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
												:class="{selected: op == x.ID}"
												v-on:click="selectOp(x.ID)"
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
