import utils from '../utils.js';

export default {
	data: function() {
		return {
			velocitySteps: 5,
			minIOU: 0.1,
			maxAge: 10,
			weight: 1.0,

			addCategoryInput: '',
		};
	},
	props: ['node'],
	created: function() {
		try {
			let s = JSON.parse(this.node.Params);
			this.velocitySteps = s.Simple.VelocitySteps;
			this.minIOU = s.Simple.MinIOU;
			this.maxAge = s.Simple.MaxAge;
			this.weight = s.Weight;
		} catch(e) {}
	},
	methods: {
		save: function() {
			let params = JSON.stringify({
				Simple: {
					VelocitySteps: parseInt(this.velocitySteps),
					MinIOU: parseFloat(this.minIOU),
					MaxAge: parseInt(this.maxAge),
				},
				Weight: parseFloat(this.weight),
			});
			utils.request(this, 'POST', '/exec-nodes/'+this.node.ID, JSON.stringify({
				Params: params,
			}), () => {
				this.$router.push('/ws/'+this.$route.params.ws+'/pipeline');
			});
		},
	},
	template: `
<div class="small-container m-2">
	<template v-if="node != null">
		<div class="form-group row">
			<label class="col-sm-4 col-form-label">Velocity Steps</label>
			<div class="col-sm-8">
				<input v-model="velocitySteps" type="text" class="form-control">
				<small class="form-text text-muted">
					The number of frames to use when estimating an object's velocity.
					For example, 5 means an object's velocity will be estimated as its displacement over 5 frames divided by 5.
					Velocity is used to estimate an object's likely current position based on its previous position.
				</small>
			</div>
		</div>
		<div class="form-group row">
			<label class="col-sm-4 col-form-label">Minimum IOU</label>
			<div class="col-sm-8">
				<input v-model="minIOU" type="text" class="form-control">
				<small class="form-text text-muted">
					IOU is the intersection-over-union area, and measures overlap between bounding boxes.
					A bounding box in a later frame must overlap with an object's estimated position by at least this much IOU to be associated with the object.
				</small>
			</div>
		</div>
		<div class="form-group row">
			<label class="col-sm-4 col-form-label">Maximum Age</label>
			<div class="col-sm-8">
				<input v-model="maxAge" type="text" class="form-control">
				<small class="form-text text-muted">
					If we have not seen an object for this many frames, we assume it is gone and remove it.
				</small>
			</div>
		</div>
		<div class="form-group row">
			<label class="col-sm-4 col-form-label">Re-identification Weight</label>
			<div class="col-sm-8">
				<input v-model="weight" type="text" class="form-control">
				<small class="form-text text-muted">
					Weight of re-identification model relative to the heuristic motion estimator.
				</small>
			</div>
		</div>
		<button v-on:click="save" type="button" class="btn btn-primary">Save</button>
	</template>
</div>
	`,
};
