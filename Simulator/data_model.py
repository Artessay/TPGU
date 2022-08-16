import os
import random
import numpy as np
import pandas as pd


class BoilerDataSet(object):
    """
    first run data_preparation.py to generate data.csv
    prepare boiler training and validation dataset
    simple version(small action dimension)

    """
    # @todo 还未归一化和随机化
    
    def __init__(self, num_steps, val_ratio=0.1):
        self.num_steps = num_steps  # 历史步长
        self.val_ratio = val_ratio  # 训练集与测试集比例
        
        # Read csv file
        csv_path = os.path.join("data", "sim_train.csv")
        self.raw_data = pd.read_csv(csv_path, index_col='时间戳')

        # 划分训练集和测试集
        self.train_X, self.train_y, self.valid_X, self.valid_y = self.prepare_data(self.raw_data)

    def prepare_data(self, data):
        # split into groups of num_steps

        # 取出输入数据，学习num_steps步长的历史，iloc：通过行号获取行数据
        X = np.array([data.iloc[i: i + self.num_steps].values
                    for i in range(len(data) - self.num_steps)])

        # 取出输出数据，预测第num_steps步的值训练，ix / loc 可以通过行号和行标签进行索引
        # 这里只要对状态量进行预测即可，0-157列为 'A磨煤机电流':'大渣可燃物含量'
        y = np.array([data.iloc[i + self.num_steps, 0:158].values
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