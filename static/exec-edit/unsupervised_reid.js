import PytorchTrainGeneric from './pytorch_train-generic.js';
export default PytorchTrainGeneric({
	disabled: ['dataset'],
	created: function() {
		if(!('TrackDuration' in this.params)) this.$set(this.params, 'TrackDuration', 4);
	},
	basicTemplate: `
<div class="small-container">
	<div class="row mb-2">
		<label class="col-sm-4 col-form-label">Track Duration</label>
		<div class="col-sm-8">
			<input v-model.number="params.TrackDuration" type="text" class="form-control">
			<small class="form-text text-muted">
				Average duration in seconds that objects are visible in the video (rough estimate).
			</small>
		</div>
	</div>
</div>
	`,
});
