Vue.component('exec-edit-model', {
	data: function() {
		return {
			trainNodes: {},
			trainNodeID: '',
		};
	},
	props: ['node'],
	created: function() {
		myCall('GET', '/train-nodes', null, (trainNodes) => {
			trainNodes.forEach((node) => {
				if(!node.Trained) {
					return;
				}
				this.$set(this.trainNodes, node.ID, node);
			});
		});
		try {
			let s = JSON.parse(this.node.Params);
			this.trainNodeID = s.TrainNodeID;
		} catch(e) {}
	},
	methods: {
		save: function() {
			let params = {
				TrainNodeID: parseInt(this.trainNodeID),
			};
			myCall('POST', '/exec-nodes/'+this.node.ID, JSON.stringify({
				Params: JSON.stringify(params),
				DataTypes: this.trainNodes[params.TrainNodeID].Outputs,
			}));
		},
	},
	template: `
<div class="small-container m-2">
	<div class="form-group row">
		<label class="col-sm-2 col-form-label">Train Node</label>
		<div class="col-sm-10">
			<select v-model="trainNodeID" class="form-control">
				<template v-for="node in trainNodes">
					<option :key="node.ID" :value="node.ID">{{ node.Name }}</option>
				</template>
			</select>
		</div>
	</div>
	<button v-on:click="save" type="button" class="btn btn-primary">Save</button>
</div>
	`,
});
