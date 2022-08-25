import tensorflow as tf
import numpy as np
import os
import random
import time
from simrnn_cell import SimulatorRNNCell

class RNNSimulatorModel(object):
    def __init__(self,
                 cell_config,
                 FLAGS):
        """ Construct simulator model using self_designed cell """
        # 一个unit对应一个神经元
        self.coaler_cell_size, self.burner_cell_size, self.steamer_cell_size = cell_config.num_units
        self.input_size = FLAGS.input_size
        self.output_size = FLAGS.output_size
        # output一定是state，所以output size == state size
        self.coaler_output_size = cell_config.coaler_state_size
        self.burner_output_size = cell_config.burner_state_size
        self.steamer_output_size = cell_config.steamer_state_size

        self.batch_size = FLAGS.batch_size
        self.n_steps = FLAGS.num_steps
        self.l2_weight = FLAGS.l2_weight
        self.grad_clip = FLAGS.grad_clip
        # 以上都是标量
        # inputs.shape = (number of examples, number of input, dimension of each input).
        """
        在代码层面，每一个tensor值在graph上都是一个op，当我们将train数据分成一个个minibatch
        然后传入网络进行训练时，每一个minibatch都将是一个op，这样的话，一副graph上的op未免太多，
        也会产生巨大的开销；于是就有了tf.placeholder()，我们每次可以将 一个minibatch传入到
        x = tf.placeholder(tf.float32,[None,32])上，下一次传入的x都替换掉上一次传入的x，
        这样就对于所有传入的minibatch x就只会产生一个op，不会产生其他多余的op，进而减少了graph的开销。
        placeholder仅限于tensorflow 1.x
        inputs是三维的，第一维不定，后面两维应该就是.csv的一部分？
        [有关placeholder](https://blog.csdn.net/hgnuxc_1993/article/details/118164675)
        """
        # inputs.shape == (batch_size, self.num_steps, table_col) == (batch_size, self.num_steps, self.input_size)
        # targets.shape == (batch_size, self.output_size)
        self.inputs = tf.compat.v1.placeholder(tf.float32, [None, self.n_steps, self.input_size], name="inputs")
        self.targets = tf.compat.v1.placeholder(tf.float32, [None, self.output_size], name="targets")
        self.learning_rate = tf.compat.v1.placeholder(tf.float32, None, name="learning_rate")
        self.keep_prob = tf.compat.v1.placeholder(tf.float32, None, name="keep_prob") #似乎是input中元素被保留下来的概率，防止过拟合

        # 创建自定义的RNNcell
        self.cell = SimulatorRNNCell(cell_config, self.keep_prob)
        """
        Run dynamic RNN
        假设num_units = [4, 2, 2], 那么state_size=((4,2,2),(4,2,2))，再假设batch_size=3，
        那么zero_state返回的就是((t1,t2,t2),(t1,t2,t2)),其中
        t1=<tf.Tensor: shape=(3, 4), dtype=float32, numpy=array([[0., 0., 0., 0.],
                                                                 [0., 0., 0., 0.],
                                                                 [0., 0., 0., 0.]], dtype=float32)>
        t2=<tf.Tensor: shape=(3, 2), dtype=float32, numpy=array([[0., 0.],
                                                                 [0., 0.],
                                                                 [0., 0.]], dtype=float32)>
        dynamic_rnn函数：
        - time_major参数： 决定了输出tensor的格式，如果为True, 张量的形状必须为 
            [max_time, batch_size,cell.output_size]。如果为False, 
            tensor的形状必须为[batch_size, max_time, cell.output_size]，cell.output_size表示rnn cell中神经元个数。
        """
        self.cell_init_state = self.cell.zero_state(self.batch_size, dtype=tf.float32)
        cell_outputs, cell_final_state = tf.nn.dynamic_rnn( # 应该就是返回call的结果(new_h, new_state)，在这里还调用了self.cell的build和call方法
            self.cell, self.inputs, initial_state=self.cell_init_state, time_major=False, scope="dynamic_rnn")
        """cell_outputs是一个三元元组，三个元素的shape分别=(1,10,256),(1,10,128),(1,10,128)
            也就是cell_outputs.get_shape() = (batch_size, num_steps, cell_size)

           cell_final_state是这样一个元组，
           LSTMStateTuple(
            c=( <tf.Tensor 'dynamic_rnn/while/Exit_5:0' shape=(1, 256) dtype=float32>, 
                <tf.Tensor 'dynamic_rnn/while/Exit_6:0' shape=(1, 128) dtype=float32>, 
                <tf.Tensor 'dynamic_rnn/while/Exit_7:0' shape=(1, 128) dtype=float32>
                ), 
           h=(  <tf.Tensor 'dynamic_rnn/while/Exit_8:0' shape=(1, 256) dtype=float32>, 
                <tf.Tensor 'dynamic_rnn/while/Exit_9:0' shape=(1, 128) dtype=float32>, 
                <tf.Tensor 'dynamic_rnn/while/Exit_10:0' shape=(1, 128) dtype=float32>
                )
            )
           """
        # self.coaler_output.shape() == (batch_size, _coaler_num_units)，以此类推
        coaler_output, burner_output, steamer_output = cell_outputs # 各个的h
        self.coaler_output = coaler_output[:, -1, :] # 【为何只取最后一步？最后一步的含义？】
        self.burner_output = burner_output[:, -1, :]
        self.steamer_output = steamer_output[:, -1, :]

        # pred应该是predict的缩写，是输出结果。需要把h做一个线性变换+sigmoid
        # tf.Variable默认trainable=True
        # pred = sigmoid(out * W + b)
        ws_out_coaler = tf.Variable(
            # truncated_normal就是生成一个N(0,1)的tensor，但生成的数据都会落在平均值±2个标准差内
            tf.random.truncated_normal([self.coaler_cell_size, self.coaler_output_size]), name="W_coaler")
        bs_out_coaler = tf.Variable(
            tf.constant(0.1, shape=[self.coaler_output_size]), name="bias_coaler")
        ws_out_burner = tf.Variable(
            tf.random.truncated_normal([self.burner_cell_size, self.burner_output_size]), name="W_burner")
        bs_out_burner = tf.Variable(
            tf.constant(0.1, shape=[self.burner_output_size]), name="bias_burner")
        ws_out_steamer = tf.Variable(
            tf.random.truncated_normal([self.steamer_cell_size, self.steamer_output_size]), name="W_steamer")
        bs_out_steamer = tf.Variable(
            tf.constant(0.1, shape=[self.steamer_output_size]), name="bias_steamer")

        # self.coaler_pred.shape() == (1, self.coaler_output_size) == (1, cell_config.coaler_state_size)，以此类推
        # tf.matmul和tf.python.ops.math_ops.matmul是一样的作用
        self.coaler_pred = tf.matmul(self.coaler_output, ws_out_coaler) + bs_out_coaler 
        self.burner_pred = tf.matmul(self.burner_output, ws_out_burner) + bs_out_burner
        self.steamer_pred = tf.matmul(self.steamer_output, ws_out_steamer) + bs_out_steamer
        self.pred = tf.concat([self.coaler_pred, self.burner_pred, self.steamer_pred], axis=1)
        self.pred = tf.sigmoid(self.pred)
        # 最后的self.pred.shape() == (1, output_size) == 
        # (1, cell_config.coaler_state_size + cell_config.burner_state_size + cell_config.steamer_state_size) 
        # self.pred_summ = tf.summary.histogram("pred", self.pred)


        # train loss，默认情况下不考虑l2_loss，因为它的权重是0
        self.tv = tf.compat.v1.trainable_variables() # 很可能是self.cell.build里面定义的那些权重
        self.l2_loss = self.l2_weight * tf.reduce_sum( 
            [tf.nn.l2_loss(v) for v in self.tv if not ("noreg" in v.name or "bias" in v.name)], name="l2_loss")
        self.mse = tf.reduce_mean(tf.square(self.pred - self.targets), name="loss_mse_train")
        self.loss = self.mse + self.l2_loss

        # gradients clip
        grads, _ = tf.clip_by_global_norm(tf.gradients(self.loss, self.tv), self.grad_clip)
        # 似乎有三种optimizer可以用
        # optimizer = tf.train.MomentumOptimizer(self.learning_rate, 0.9)
        # optimizer = tf.train.RMSPropOptimizer(self.learning_rate)
        optimizer = tf.compat.v1.train.AdamOptimizer(self.learning_rate)
        self.train_opt = optimizer.apply_gradients(zip(grads, self.tv)) 
        # 上面zip的意思是形成一系列元组：[(grad[0],self.tv[0]), (grad[1],self.tv[1]), ...]，
        # 这个list的len是min(len(grad), len(self.tv))，元素多的那一个list后面的元素就不要了

        # summary
        self.loss_summ = tf.compat.v1.summary.scalar("loss_mse_train", self.loss)
        self.learning_rate_summ = tf.compat.v1.summary.scalar("learning_rate", self.learning_rate)
        # for var in tf.trainable_variables():
        #     tf.summary.histogram(var.name, var)
        self.merged_summ = tf.compat.v1.summary.merge_all()

