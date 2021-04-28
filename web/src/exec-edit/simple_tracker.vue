<template>
<div class="small-container m-2">
	<template v-if="node != null">
		<div class="form-group row">
			<label class="col-sm-2 col-form-label">Velocity Steps</label>
			<div class="col-sm-10">
				<input v-model="velocitySteps" type="text" class="form-control">
			</div>
		</div>
		<div class="form-group row">
			<label class="col-sm-2 col-form-label">Minimum IOU</label>
			<div class="col-sm-10">
				<input v-model="minIOU" type="text" class="form-control">
			</div>
		</div>
		<div class="form-group row">
			<label class="col-sm-2 col-form-label">Maximum Age</label>
			<div class="col-sm-10">
				<input v-model="maxAge" type="text" class="form-control">
			</div>
		</div>
		<button v-on:click="save" type="button" class="btn btn-primary">Save</button>
	</template>
</div>
</template>

<script>
import utils from '../utils.js';

export default {
	data: function() {
		return {
			velocitySteps: 5,
			minIOU: 0.1,
			maxAge: 10,

			addCategoryInput: '',
		};
	},
	props: ['node'],
	created: function() {
		try {
			let s = JSON.parse(this.node.Params);
			this.velocitySteps = s.VelocitySteps;
			this.minIOU = s.MinIOU;
			this.maxAge = s.MaxAge;
		} catch(e) {}
	},
	methods: {
		save: function() {
			let params = JSON.stringify({
				VelocitySteps: parseInt(this.velocitySteps),
				MinIOU: parseFloat(this.minIOU),
				MaxAge: parseInt(this.maxAge),
			});
			utils.request(this, 'POST', '/exec-nodes/'+this.node.ID, JSON.stringify({
				Params: params,
			}), () => {
				this.$router.push('/ws/'+this.$route.params.ws+'/pipeline');
			});
		},
	},
};
</script>