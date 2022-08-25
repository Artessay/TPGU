import numpy as np
import os
import pandas as pd
import random


class BoilerDataSet(object):
    """
    first run data_preparation.py to generate data.csv
    prepare boiler training and validation dataset
    simple version(small action dimension)

    """
    def __init__(self, num_steps, output_size, val_ratio=0.1):
        # num_steps: 一个步长，见_prepare_data
        # val_ratio: 有多少比例的数据是用来测试的
        self.num_steps = num_steps
        self.val_ratio = val_ratio

        # Read csv file，index_col作为主码，如果不设置的话默认从0开始设置id
        # raw_seq是一个DataFrame, 类似一个字典，比如raw_seq['A磨煤机电流']就会得到那一列
        self.raw_seq = pd.read_csv(os.path.join(".\\Simulator\\data", "sample_data_resample_1T.csv"), index_col='时间戳')
        # sort csv file
        cols = self.raw_seq.columns.tolist()
        # print("origin len: {0}".format(len(cols)))
        cols = (cols[51:52] + cols[53:59] + cols [60:61] + cols[62:63] + cols[150:152]   # external input 
            + cols[0:50] + cols[52:53] + cols[122:139]  # Coal Pulverizing state
            + cols[50:51] + cols[59:60] + cols[61:62] + cols[63:101] + cols[112:114] + cols[118:122] + cols[139:145] + cols[146:149] + cols[152:158]    # Burning state
            + cols[101:112] + cols[114:118] + cols[145:146] + cols[149:150] # Steam Circulation state
            + cols[158:173] + cols[196:202] # Coal Pulverizing action
            + cols[173:192]                 # Burning action
            + cols[192:196])                # Steam Circulation action
        self.raw_seq = self.raw_seq[cols]
        self.train_X, self.train_y, self.val_X, self.val_y = self._prepare_data(self.raw_seq, output_size)

    def _prepare_data(self, seq, output_size):
        # split into groups of num_steps
        # iloc函数：通过行号（从0开始）来取行数据
        # X中一个元素就是10行数据，按照0-9,1-10,2-11，...来取数据
        # （.values就是不把主码算在里面的数据！）
        X = np.array([seq.iloc[i: i + self.num_steps].values
                      for i in range(len(seq) - self.num_steps)])
        # seq.ix在高版本的Pandas已经无法使用，loc可以替代
        # 可以选定一块区域（指定行列的范围），这里选定了i + self.num_steps一行，选中了一些列
        # 注意到len(y) == len(X)
        y = np.array([seq.iloc[i + self.num_steps, 0:output_size].values
                      for i in range(len(seq) - self.num_steps)])
        # X.shape==(len(seq) - self.num_steps, self.num_steps, input_size)
        # y.shape==(len(seq) - self.num_steps, output_size)

        # 训练的大小
        train_size = int(len(X) * (1.0 - self.val_ratio))
        train_X, val_X = X[:train_size], X[train_size:]
        train_y, val_y = y[:train_size], y[train_size:]
        # train_X.shape==(train_size, self.num_steps, input_size)
        # train_y.shape==(train_size, output_size)
        return train_X, train_y, val_X, val_y

    def generate_one_epoch(self, data_X, data_y, batch_size):
        num_batches = int(len(data_X)) // batch_size
        # if batch_size * num_batches < len(self.train_X):
        #     num_batches += 1

        batch_indices = list(range(num_batches))
        random.shuffle(batch_indices) # 打乱顺序
        for j in batch_indices:
            batch_X = data_X[j * batch_size: (j + 1) * batch_size]
            batch_y = data_y[j * batch_size: (j + 1) * batch_size]
            yield batch_X, batch_y 
            # yield就是不断返回东西，最后在调用函数的地方可以收到一个迭代器，
            # 遍历即可拿到for循环的每一次返回值
            # 每一次的返回值形状如下：
            # batch_X.shape=(batch_size, self.num_steps, table_col)
            # batch_y.shape=(batch_size, table_col2)
            # 其中table_col, table_col2见上面定义
