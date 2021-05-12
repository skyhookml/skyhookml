<template>
<div class="el-high flex-x-container">
	<div class="flex-content flex-container">
		<div class="flex-container" v-if="node != null">
			<textarea v-model="code" v-on:keydown="autoindent($event)" class="el-big" placeholder="Your Code Here"></textarea>
		</div>
		<div class="my-1">
			<p>See examples on <a href="https://www.skyhookml.org/docs/python/">skyhookml.org</a>.</p>
		</div>
		<div class="my-1 flex-x-container">
			<div class="me-1 flex-content">
				<button v-on:click="save" type="button" class="btn btn-primary btn-sm el-wide">Save</button>
			</div>
			<div class="ms-1">
				<button
					v-on:click="generateTemplate"
					type="button"
					class="btn btn-warning btn-sm el-wide"
					data-toggle="tooltip"
					data-placement="bottom"
					title="Generate boilerplate code based on the currently configured inputs and specified outputs of this operation."
					>
					Generate Template
				</button>
			</div>
		</div>
	</div>
	<div class="ms-2">
		<h4>Outputs</h4>
		<p>Define the outputs of this node.</p>
		<table class="table">
			<thead>
				<tr>
					<th>Name</th>
					<th>Type</th>
					<th></th>
				</tr>
			</thead>
			<tbody>
				<tr v-for="(output, i) in outputs" :key="output.Name">
					<td>{{ output.Name }}</td>
					<td>{{ $globals.dataTypes[output.DataType] }}</td>
					<td>
						<button type="button" class="btn btn-danger" v-on:click="removeOutput(i)">Remove</button>
					</td>
				</tr>
				<tr>
					<td>
						<input type="text" class="form-control" v-model="addOutputForm.name" />
					</td>
					<td>
						<select v-model="addOutputForm.dataType" class="form-select">
							<option v-for="(name, dt) in $globals.dataTypes" :value="dt">{{ name }}</option>
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
</template>

<script>
import utils from '../utils.js';

export default {
	data: function() {
		return {
			addOutputForm: null,
			code: '',
			outputs: [],
		};
	},
	props: ['node'],
	created: function() {
		try {
			let params = JSON.parse(this.node.Params);
			this.code = params.Code;
			this.outputs = params.Outputs;
		} catch(e) {}
		this.resetForm();
	},
	methods: {
		autoindent: function(e) {
			var el = e.target;
			if(e.keyCode == 9) {
				// tab
				e.preventDefault();
				var start = el.selectionStart;
				var prefix = this.code.substring(0, start);
				var suffix = this.code.substring(el.selectionEnd);
				this.code = prefix + '\t' + suffix;

				Vue.nextTick(function() {
					el.selectionStart = start+1;
					el.selectionEnd = start+1;
				});
			} else if(e.keyCode == 13) {
				// enter
				e.preventDefault();
				var start = el.selectionStart;
				var prefix = this.code.substring(0, start);
				var suffix = this.code.substring(el.selectionEnd);
				var prevLine = prefix.lastIndexOf('\n');

				var spacing = '';
				if(prevLine != -1) {
					for(var i = prevLine+1; i < start; i++) {
						var char = this.code[i];
						if(char != '\t' && char != ' ') {
							break;
						}
						spacing += char;
					}
				}
				this.code = prefix + '\n' + spacing + suffix;
				Vue.nextTick(function() {
					el.selectionStart = start+1+spacing.length;
					el.selectionEnd = el.selectionStart;
				});
			}
		},

		generateTemplate: function() {
			if(!this.node.Parents || this.node.Parents['inputs'].length == 0) {
				alert('Before generating template, you must configure the inputs to this node (go to Pipeline, select node, and add inputs).');
				return;
			}
			let inputTypes = this.node.Parents['inputs'].map((parent) => parent.DataType);
			let outputs = this.node.Outputs;
			if(!outputs) {
				outputs = [];
			}

			let typeHelpTexts = {
				'image': {
					'per_frame': ['Data: A numpy array with dimensions (width, height, 3).', 'Metadata: N/A.'],
					'all': ['Data: A numpy array with dimensions (1, width, height, 3).', 'Metadata: N/A.'],
				},
				'video': {
					'per_frame': ['Data: A numpy array with dimensions (width, height, 3).', 'Metadata: N/A.'],
					'all': ['Data: A numpy array with dimensions (nframes, width, height, 3).', 'Metadata: N/A.'],
				},
				'detection': [
					'Data: Object detections: a list (or list of lists) of bounding boxes.',
					'Each detection has keys Left, Top, Right, Bottom, and optionally Category, TrackID, Score, Metadata.',
					'Metadata: Optional keys CanvasDims and Categories.',
					'Example: {"Detections": [{"Left": 100, "Right": 150, "Top": 300, "Bottom": 350}], "Metadata": {"CanvasDims": [1280, 720]}}',
				],
				'shape': [
					'Data: Shapes: a list (or list of lists) of shapes.',
					'Each shape has keys Type, Points, and optionally Category, TrackID, Metadata.',
					'Metadata: Optional keys CanvasDims and Categories.',
					'Example: {"Shapes": [{"Type": "point", Points: [[100, 100]]}], "Metadata": {"CanvasDims": [1280, 720]}}',
				],
				'int': [
					'Data: A list of integers, or a single integer.',
					'Metadata: Optional key Categories.',
					'Example: {"Ints": 2, "Metadata": {"Categories": ["person", "car", "giraffe"]}}',
				],
				'floats': {
					'per_frame': ['Data: A list of floats.', 'Metadata: N/A.'],
					'all': ['Data: A list of lists of floats.', 'Metadata: N/A.'],
				},
				'string': {
					'per_frame': ['Data: A string.', 'Metadata: N/A.'],
					'all': ['Data: A list of strings.', 'Metadata: N/A.'],
				},
				'array': {
					'per_frame': ['Data: A numpy array with dimensions (width, height, channels).', 'Metadata: A dict with keys Width, Height, Channels, Type.'],
					'all': ['Data: A numpy array with dimensions (length, width, height, channels).', 'Metadata: A dict with keys Width, Height, Channels, Type.'],
				},
				'table': [
					'Data: A list of list of strings, where each sub-list corresponds to the values in one row.',
					'Metadata: A list of each columns, where each column is specified by a dict with keys Label, Type.',
					'Example: {"Specs": [{"Label": "Column 1", "Type": "string"}], "Data": [["Row 1"], ["Row 2"]]}',
				],
			};
			let tmpl = `from skyhook.op import per_frame, all_decorate
# Template for inputting data one element at a time.
# Only works for sequence data types.
@per_frame
def f(ARGLIST):
PER_FRAME_DOC
	pass

# Template for inputting entire data item at once.
@all_decorate
def f(ARGLIST):
ALL_DOC
	pass`;

			// helper function to get help text given a data type
			// mode is either 'per_frame' or 'all'
			let getHelpText = function(name, dt, mode) {
				let helpText = typeHelpTexts[dt];
				if(typeof helpText === 'object' && helpText[mode]) {
					helpText = helpText[mode];
				}
				if(!Array.isArray(helpText)) {
					helpText = [helpText];
				}
				let text = '\t- '+name+': ' + helpText[0] + '\n';
				for(let el of helpText.slice(1)) {
					text += '\t'+el + '\n';
				}
				return text;
			};

			// assign variable names to the types
			let variableNames = [];
			let usedNames = {};
			for(let dt of inputTypes) {
				let name;
				if(dt in usedNames) {
					name = dt+(usedNames[dt]+1);
					usedNames[dt]++;
				} else {
					name = dt+'0';
					usedNames[dt] = 0;
				}
				variableNames.push(name);
			}

			// helper function to get doc string part for one mode (per_frame/all)
			let getDoc = function(mode) {
				let doc = "\t'''\n\tInputs (each input is a dict with keys 'Data' and 'Metadata'):\n";
				for(let i = 0; i < inputTypes.length; i++) {
					doc += getHelpText(variableNames[i], inputTypes[i], mode);
				}
				doc += "\tReturns: a tuple where elements are either data only or a dict with keys 'Data' and 'Metadata'\n"
				for(let i = 0; i < outputs.length; i++) {
					let name = 'Index ' + i + ' (' + outputs[i].Name + ')';
					doc += getHelpText(name, outputs[i].DataType, mode);
				}
				doc += "\t'''";
				return doc;
			}

			// update template
			tmpl = tmpl.replaceAll('ARGLIST', variableNames.join(', '));
			tmpl = tmpl.replace('PER_FRAME_DOC', getDoc('per_frame'));
			tmpl = tmpl.replace('ALL_DOC', getDoc('all'));

			this.code = tmpl;
		},

		// modifying outputs
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

		save: function() {
			let params = {
				Code: this.code,
				Outputs: this.outputs,
			};
			utils.request(this, 'POST', '/exec-nodes/'+this.node.ID, JSON.stringify({
				Params: JSON.stringify(params),
			}), () => {
				this.$router.push('/ws/'+this.$route.params.ws+'/pipeline');
			});
		},
	},
};
</script>
