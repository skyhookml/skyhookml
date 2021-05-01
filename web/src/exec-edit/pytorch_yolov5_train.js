import PytorchTrainGeneric from './pytorch_train-generic.js';
import SelectInputSize from './select-input-size.vue';

export default PytorchTrainGeneric({
	components: {
		'select-input-size': SelectInputSize,
	},
	disabled: ['model', 'dataset'],
	created: function() {
		if(!('Mode' in this.params)) this.$set(this.params, 'Mode', 'x');
		if(!('ValPercent' in this.params)) this.$set(this.params, 'ValPercent', 20);
		if(!('Resize' in this.params)) {
			this.$set(this.params, 'Resize', {
				Mode: 'scale-down',
				MaxDimension: 640,
				Width: 416,
				Height: 416,
				Multiple: 32,
			});
		}
	},
	basicTemplate: `
<div class="small-container">
	<div class="row mb-2">
		<label class="col-sm-4 col-form-label">Mode</label>
		<div class="col-sm-8">
			<select v-model="params.Mode" class="form-select">
				<option value="s">YOLOv5s</option>
				<option value="m">YOLOv5m</option>
				<option value="l">YOLOv5l</option>
				<option value="x">YOLOv5x</option>
			</select>
			<small class="form-text text-muted">
				Choose a model size: small (s), medium (m), large (l), and x-large (x).
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
