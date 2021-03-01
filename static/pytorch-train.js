import utils from './utils.js';

export default {
	data: function() {
		return {
			p: {},
		};
	},
	props: ['node', 'value'],
	created: function() {
		let p = {};
		try {
			let s = JSON.parse(this.value);
			p = s;
		} catch(e) {}
		if(!p.LearningRate) {
			p.LearningRate = 0.001;
		}
		if(!p.Optimizer) {
			p.Optimizer = 'adam';
		}
		if(!p.BatchSize) {
			p.BatchSize = 1;
		}
		if(!p.StopCondition) {
			p.StopCondition = {
				MaxEpochs: 0,
				ScoreEpsilon: 0,
				ScoreMaxEpochs: 25,
			}
		}
		if(!p.ModelSaver) {
			p.ModelSaver = {
				Mode: 'best',
			};
		}
		if(!p.RateDecay) {
			p.RateDecay = {
				Op: '',
			}
		}
		this.p = p;
		this.update();
	},
	methods: {
		update: function() {
			this.$emit('input', JSON.stringify(this.p));
		},
	},
	template: `
<div class="small-container">
	<h3>Basics</h3>
	<div class="form-group row">
		<label class="col-sm-2 col-form-label">Learning Rate</label>
		<div class="col-sm-10">
			<input v-model.number="p.LearningRate" type="text" class="form-control" @change="update">
		</div>
	</div>
	<div class="form-group row">
		<label class="col-sm-2 col-form-label">Optimizer</label>
		<div class="col-sm-10">
			<select v-model="p.Optimizer" class="form-control" @change="update">
				<option value="adam">Adam</option>
			</select>
		</div>
	</div>
	<div class="form-group row">
		<label class="col-sm-2 col-form-label">Batch Size</label>
		<div class="col-sm-10">
			<input v-model.number="p.BatchSize" type="text" class="form-control" @change="update">
		</div>
	</div>

	<h3>Stop Condition</h3>
	<div class="form-group row">
		<label class="col-sm-2 col-form-label">Max Epochs</label>
		<div class="col-sm-10">
			<input v-model.number="p.StopCondition.MaxEpochs" type="text" class="form-control" @change="update">
			<small class="form-text text-muted">
				Stop training after this many epochs. Leave 0 to disable this stop condition.
			</small>
		</div>
	</div>
	<div class="form-group row">
		<label class="col-sm-2 col-form-label">Epochs Without Improvement</label>
		<div class="col-sm-10">
			<input v-model.number="p.StopCondition.ScoreMaxEpochs" type="text" class="form-control" @change="update">
			<small class="form-text text-muted">
				Stop training if this many epochs have elapsed without non-negligible improvement in the score. Leave 0 to disable this stop condition.
			</small>
		</div>
	</div>
	<div class="form-group row">
		<label class="col-sm-2 col-form-label">Improvement Threshold</label>
		<div class="col-sm-10">
			<input v-model.number="p.StopCondition.ScoreEpsilon" type="text" class="form-control" @change="update">
			<small class="form-text text-muted">
				Increases in the score less than this threshold are considered negligible. 0 implies that any increase will reset the timer for Epochs Without Improvement.
			</small>
		</div>
	</div>

	<h3>Model Saver</h3>
	<div class="form-group row">
		<label class="col-sm-2 col-form-label">Saver Mode</label>
		<div class="col-sm-10">
			<select v-model="p.ModelSaver.Mode" class="form-control" @change="update">
				<option value="latest">Save the latest model</option>
				<option value="best">Save the model with best validation score</option>
			</select>
		</div>
	</div>

	<h3>Rate Decay</h3>
	<p>TODO</p>
</div>
	`,
};
