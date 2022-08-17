# %%
import random
import numpy as np
import pandas as pd
from os import path, mkdir

# %%
import tensorflow
tf = tensorflow.compat.v1

tf.disable_eager_execution()
tf.experimental.output_all_intermediates(True)

# %% [markdown]
# 定义参数

# %%
num_steps = 10
valid_ratio = 0.2

input_size = 202
num_neurons = 160
num_layers = 3
output_size = 158

learning_rate = 0.001
learning_rate_decay = 0.95

max_epoch = 50
batch_size = 1

save_log_iter = 10
display_iter = 20

# %% [markdown]
# 数据集处理类：

# %%
class BoilerDataSet(object):
    """
    first run data_preparation.py to generate data.csv
    prepare boiler training and validation dataset
    simple version(small action dimension)

    """
    
    def __init__(self, num_steps, val_ratio=0.1):
        self.num_steps = num_steps  # 历史步长
        self.val_ratio = val_ratio  # 训练集与测试集比例
        
        # Read csv file
        self.raw_data = pd.read_csv("./Simulator/data/sim_train.csv", index_col='时间戳')

        # sort csv file
        cols = self.raw_data.columns.tolist()
        # print("origin len: {0}".format(len(cols)))
        cols = (cols[51:52] + cols[53:59] + cols [60:61] + cols[62:63] + cols[150:152]   # external input 
            + cols[0:50] + cols[52:53] + cols[122:139]  # Coal Pulverizing state
            + cols[50:51] + cols[59:60] + cols[61:62] + cols[63:101] + cols[112:114] + cols[118:122] + cols[139:145] + cols[146:149] + cols[152:158]    # Burning state
            + cols[101:112] + cols[114:118] + cols[145:146] + cols[149:150] # Steam Circulation state
            + cols[158:173] + cols[196:202] # Coal Pulverizing action
            + cols[173:192]                 # Burning action
            + cols[192:196])                # Steam Circulation action
        print("ordered len: {0}".format(len(cols)))
        # self.raw_data = self.raw_data[cols]

        # 划分训练集和测试集
        self.train_X, self.train_y, self.valid_X, self.valid_y = self.prepare_data(self.raw_data)

    def prepare_data(self, data):
        # split into groups of num_steps

        # 取出输入数据，学习num_steps步长的历史，iloc：通过行号获取行数据
        X = np.array([data.iloc[i: i + self.num_steps].values
                    for i in range(len(data) - self.num_steps)])

        # 取出输出数据，预测第num_steps步的值训练，ix / loc 可以通过行号和行标签进行索引
        # 这里只要对状态量进行预测即可，0-157列为 'A磨煤机电流':'大渣可燃物含量'
        y = np.array([data.iloc[i+1: i + self.num_steps+1, 0:158].values
                    for i in range(len(data) - self.num_steps)])

        train_size = int(len(X) * (1.0 - self.val_ratio))
        train_X, valid_X = X[:train_size], X[train_size:]
        train_y, valid_y = y[:train_size], y[train_size:]
        return train_X, train_y, valid_X, valid_y

    def generate_one_epoch(self, data_X, data_y, batch_size):
        num_batches = int(len(data_X)) // batch_size
        # if batch_size * num_batches < len(self.train_X):
        #     num_batches += 1

        batch_indices = list(range(num_batches))
        random.shuffle(batch_indices)
        for j in batch_indices:
            batch_X = data_X[j * batch_size: (j + 1) * batch_size]
            batch_y = data_y[j * batch_size: (j + 1) * batch_size]
            yield batch_X, batch_y

# %% [markdown]
# 读入数据

# %%
# read data
boiler_dataset = BoilerDataSet(num_steps=num_steps, val_ratio=valid_ratio)
train_X, train_y = boiler_dataset.train_X, boiler_dataset.train_y
valid_X, valid_y = boiler_dataset.valid_X, boiler_dataset.valid_y

# %% [markdown]
# 在我们的示例中，一共提供了20组数据，设置的时间步长为10.因此，分别有从[0:9]->[10], [1:10]->[11], ... , [9:18]->[19] 共十组（X，y）\\
# 其中，我们训练集和测试集的比例为8：2，所以其中训练集有8组，测试集有2组。\\
# train_X (8, 10, 202) 分别为训练集组数、历史步长、数据维度 valid_y(2, 158) 分别为测试集组数和数据维度

# %%
# 打印数据信息
print('train samples: {0}'.format(len(train_X)))
print('valid samples: {0}'.format(len(valid_X)))

# %% [markdown]
# 在我们的数据中，环境变量有11个，磨煤环节共有89个变量（68个状态和21个动作）、燃烧环节共有81个变量（62个状态和19个动作）、蒸汽循环环节共有21个变量（17个状态和4个动作）
# 
# 统计可知，一共有68+62+17=147个状态，21+19+4=44个动作，加上11个外界环境变量，共202个变量

# %%
# to make this notebook's output stable across runs
def reset_graph(seed=2022):
    tf.reset_default_graph()
    tf.set_random_seed(seed)
    np.random.seed(seed)

# %% [markdown]
# 定义模型

# %%
reset_graph()

X = tf.placeholder(tf.float32, [None, num_steps, input_size])
y = tf.placeholder(tf.float32, [None, num_steps, output_size])

# basic_cell = tensorflow.keras.layers.LSTM(units=num_neurons, activation='tanh', return_sequences=True)
lstm_cells = [tf.nn.rnn_cell.BasicLSTMCell(num_units=num_neurons)
              for layer in range(num_layers)]
multi_cell = tf.nn.rnn_cell.MultiRNNCell(lstm_cells)
rnn_outputs, states = tf.nn.dynamic_rnn(multi_cell, X, dtype=tf.float32)
# tensorflow.keras.layers.RNN(cell=basic_cell)

stacked_rnn_outputs = tf.reshape(rnn_outputs, [-1, num_neurons])
stacked_outputs = tf.layers.dense(stacked_rnn_outputs, output_size)
# stacked_outputs = tensorflow.keras.layers.Dense(stacked_rnn_outputs, output_size)

outputs = tf.reshape(stacked_outputs, [-1, num_steps, output_size])

# %%
model_name = "LSTM"
logdir = './Simulator/logs/{}-{}-{}-{:.4f}/'.format(
    model_name, num_neurons, num_steps, learning_rate)
model_dir = logdir + 'saved_models/'

# 创建保存结果的文件夹
if not path.exists('./Simulator/logs'):
    mkdir('./Simulator/logs')
if not path.exists(logdir):
    mkdir(logdir)
if not path.exists(model_dir):
    mkdir(model_dir)

# %%
loss = tf.reduce_mean(tf.square(outputs - y))
optimizer = tf.train.AdamOptimizer(learning_rate= learning_rate)
training_optimizer = optimizer.minimize(loss)

loss_summary = tf.summary.scalar("loss_mse_train", loss)
merged_summary = tf.summary.merge_all()

# %%
summary_writer = tf.summary.FileWriter(logdir)
saver = tf.train.Saver()

with tf.Session() as sess:
    sess.run(tf.global_variables_initializer())     # 初始化全局变量
    
    iteration = 0           # 迭代数
    valid_losses = [np.inf] # 损失值集合

    for epoch in range(max_epoch):
        print('----------epoch {}-----------'.format(epoch))
        
        for batch_X, batch_y in boiler_dataset.generate_one_epoch(train_X, train_y, batch_size):
            iteration += 1
            sess.run(training_optimizer, feed_dict={X: batch_X, y: batch_y})
            
            summary = sess.run(merged_summary, feed_dict={X: batch_X, y: batch_y})

            if iteration % save_log_iter == 0:
                summary_writer.add_summary(summary, iteration)
            
            if iteration % display_iter == 0:
                valid_loss = 0
                for valid_batch_X, valid_batch_y in boiler_dataset.generate_one_epoch(valid_X, valid_y, batch_size):
                    batch_mse = loss.eval(feed_dict={X: valid_batch_X, y: valid_batch_y})
                    batch_loss = batch_mse      # @todo l2 loss
                    valid_loss += batch_loss
                num_batches = int(len(valid_X)) // batch_size
                valid_loss /= num_batches       # 平均每个批次的损失
                valid_losses.append(valid_loss)
                valid_loss_sum = tf.Summary(
                    value=[tf.Summary.Value(tag="valid_loss", simple_value=valid_loss)])
                summary_writer.add_summary(valid_loss_sum, iteration)

                if valid_loss < min(valid_losses[:-1]):
                    print('iter {}\tvalid_loss = {:.6f}\tmodel saved'.format(
                        iteration, valid_loss))
                    saver.save(sess, model_dir +
                                'model_{}.ckpt'.format(iteration))
                    saver.save(sess, model_dir + 'final_model.ckpt')
                else:
                    print('iter {}\tvalid_loss = {:.6f}\t'.format(
                        iteration, valid_loss))

        mse = loss.eval(feed_dict={X: batch_X, y: batch_y})
        print("epoch: ", epoch, "\tMSE: ", mse)

summary_writer.flush()
summary_writer.close()