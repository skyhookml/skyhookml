import PytorchTrainGeneric from './pytorch_train-generic.js';
import SelectInputSize from './select-input-size.vue';

export default PytorchTrainGeneric({
	components: {
		'select-input-size': SelectInputSize,
	},
	disabled: ['model', 'dataset'],
	created: function() {
		if(!('NumClasses' in this.params)) this.$set(this.params, 'NumClasses', 0);
		if(!('ValPercent' in this.params)) this.$set(this.params, 'ValPercent', 20);
		if(!('Resize' in this.params)) {
			this.$set(this.params, 'Resize', {
				Mode: 'keep',
				MaxDimension: 640,
				Width: 256,
				Height: 256,
				Multiple: 8,
			});
		}
	},
	basicTemplate: `
<div class="small-container">
	<div class="row mb-2">
		<label class="col-sm-4 col-form-label">Number of Classes</label>
		<div class="col-sm-8">
			<input v-model.number="params.NumClasses" type="text" class="form-control">
			<small class="form-text text-muted">
				The number of segmentation categories.
			</small>
		</div>
	</div>
	<div class="row mb-2">
		<label class="col-sm-4 col-form-label">Validation Percentage</label>
		<div class="col-sm-8">
			<input v-model.number="params.ValPercent" type="text" class="form-control">
			<small class="form-text text-muted">
				Use this percentage of the input data for validation. The rest will be used for training.
			</small>
		</div>
	</div>
	<hr />
	<select-input-size v-model="params.Resize"></select-input-size>
</div>
	`,
});
