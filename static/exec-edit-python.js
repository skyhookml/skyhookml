import utils from './utils.js';

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
	template: `
<div class="el-high flex-x-container">
	<div class="flex-content flex-container">
		<div class="flex-container" v-if="node != null">
			<textarea v-model="code" v-on:keydown="autoindent($event)" class="el-big" placeholder="Your Code Here"></textarea>
		</div>
		<div class="m-1">
			<button v-on:click="save" type="button" class="btn btn-primary btn-sm el-wide">Save</button>
		</div>
	</div>
	<div>
		<h4>Outputs</h4>
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
	`,
};
